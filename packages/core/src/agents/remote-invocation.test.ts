/**
 * @license
 * Copyright 2025 Google LLC
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  describe,
  it,
  expect,
  vi,
  beforeEach,
  afterEach,
  type Mock,
} from 'vitest';
import { RemoteAgentInvocation, ADCHandler } from './remote-invocation.js';
import { A2AClientManager } from './a2a-client-manager.js';
import type { RemoteAgentDefinition } from './types.js';
import type { Task } from '@a2a-js/sdk';

// Mock A2AClientManager
vi.mock('./a2a-client-manager.js', () => {
  const A2AClientManager = {
    getInstance: vi.fn(),
  };
  return { A2AClientManager };
});

// Mock ADCHandler to check it's passed correctly
vi.mock('./remote-invocation.js', async (importOriginal) => {
  const actual =
    await importOriginal<typeof import('./remote-invocation.js')>();
  return {
    ...actual,
    ADCHandler: vi.fn().mockImplementation(() => ({
      headers: vi.fn().mockResolvedValue({}),
      shouldRetryWithHeaders: vi.fn().mockResolvedValue({}),
    })),
  };
});

describe('RemoteAgentInvocation', () => {
  const mockDefinition: RemoteAgentDefinition = {
    name: 'test-agent',
    kind: 'remote',
    agentCardUrl: 'http://test-agent/card',
    displayName: 'Test Agent',
    description: 'A test agent',
    inputConfig: {
      inputs: {},
    },
  };

  const mockClientManager = {
    getClient: vi.fn(),
    loadAgent: vi.fn(),
    sendMessage: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    (A2AClientManager.getInstance as Mock).mockReturnValue(mockClientManager);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('Constructor Validation', () => {
    it('accepts valid input with string query', () => {
      expect(() => {
        new RemoteAgentInvocation(mockDefinition, { query: 'valid' });
      }).not.toThrow();
    });

    it('throws if query is missing', () => {
      expect(() => {
        new RemoteAgentInvocation(mockDefinition, {});
      }).toThrow("requires a string 'query' input");
    });

    it('throws if query is not a string', () => {
      expect(() => {
        new RemoteAgentInvocation(mockDefinition, { query: 123 });
      }).toThrow("requires a string 'query' input");
    });
  });

  describe('Execution Logic', () => {
    it('should lazy load the agent with ADCHandler if not present', async () => {
      mockClientManager.getClient.mockReturnValue(undefined);
      mockClientManager.sendMessage.mockResolvedValue({
        result: { kind: 'message', parts: [{ kind: 'text', text: 'Hello' }] },
      });

      const invocation = new RemoteAgentInvocation(mockDefinition, {
        query: 'hi',
      });
      await invocation.execute(new AbortController().signal);

      expect(mockClientManager.loadAgent).toHaveBeenCalledWith(
        'test-agent',
        'http://test-agent/card',
        expect.any(ADCHandler),
      );
    });

    it('should not load the agent if already present', async () => {
      mockClientManager.getClient.mockReturnValue({});
      mockClientManager.sendMessage.mockResolvedValue({
        result: { kind: 'message', parts: [{ kind: 'text', text: 'Hello' }] },
      });

      const invocation = new RemoteAgentInvocation(mockDefinition, {
        query: 'hi',
      });
      await invocation.execute(new AbortController().signal);

      expect(mockClientManager.loadAgent).not.toHaveBeenCalled();
    });

    it('should maintain contextId and taskId across calls', async () => {
      mockClientManager.getClient.mockReturnValue({});

      // First call return values
      mockClientManager.sendMessage.mockResolvedValueOnce({
        result: {
          kind: 'message',
          parts: [{ kind: 'text', text: 'Response 1' }],
        },
        contextId: 'ctx-1',
        taskId: 'task-1',
      });

      const invocation = new RemoteAgentInvocation(mockDefinition, {
        query: 'first',
      });

      // Execute first time
      const result1 = await invocation.execute(new AbortController().signal);
      expect(result1.returnDisplay).toBe('Response 1');
      expect(mockClientManager.sendMessage).toHaveBeenLastCalledWith(
        'test-agent',
        'first',
        { contextId: undefined, taskId: undefined },
      );

      // Prepare for second call with simulated state persistence
      mockClientManager.sendMessage.mockResolvedValueOnce({
        result: {
          kind: 'message',
          parts: [{ kind: 'text', text: 'Response 2' }],
        },
        contextId: 'ctx-1',
        taskId: 'task-2',
      });

      const result2 = await invocation.execute(new AbortController().signal);
      expect(result2.returnDisplay).toBe('Response 2');

      expect(mockClientManager.sendMessage).toHaveBeenLastCalledWith(
        'test-agent',
        'first', // Params same (re-execution of same invocation object)
        { contextId: 'ctx-1', taskId: 'task-1' }, // Used state from first call
      );

      // Third call: Task completes
      mockClientManager.sendMessage.mockResolvedValueOnce({
        result: {
          kind: 'task',
          status: { state: 'completed' },
          parts: [{ kind: 'text', text: 'Done' }],
        } as unknown as Task, // Cast because 'completed' is not in strict TaskState type
        contextId: 'ctx-1',
        taskId: 'task-1', // ID is still returned by server
      });

      const result3 = await invocation.execute(new AbortController().signal);
      expect(result3.returnDisplay).toBe('Done');

      // Fourth call: Should start new task (taskId undefined)
      mockClientManager.sendMessage.mockResolvedValueOnce({
        result: { kind: 'message', parts: ['New Task'] },
      });

      await invocation.execute(new AbortController().signal);

      expect(mockClientManager.sendMessage).toHaveBeenLastCalledWith(
        'test-agent',
        'first',
        { contextId: 'ctx-1', taskId: undefined }, // taskId cleared!
      );
    });

    it('should handle errors gracefully', async () => {
      mockClientManager.getClient.mockReturnValue({});
      mockClientManager.sendMessage.mockRejectedValue(
        new Error('Network error'),
      );

      const invocation = new RemoteAgentInvocation(mockDefinition, {
        query: 'hi',
      });
      const result = await invocation.execute(new AbortController().signal);

      expect(result.error).toBeDefined();
      expect(result.error?.message).toContain('Network error');
      expect(result.returnDisplay).toContain('Network error');
    });

    it('should use a2a helpers for extracting text', async () => {
      mockClientManager.getClient.mockReturnValue({});
      // Mock a complex message part that needs extraction
      mockClientManager.sendMessage.mockResolvedValue({
        result: {
          kind: 'message',
          parts: [
            { kind: 'text', text: 'Extracted text' },
            { kind: 'data', data: { foo: 'bar' } },
          ],
        },
      });

      const invocation = new RemoteAgentInvocation(mockDefinition, {
        query: 'hi',
      });
      const result = await invocation.execute(new AbortController().signal);

      // Just check that text is present, exact formatting depends on helper
      expect(result.returnDisplay).toContain('Extracted text');
    });
  });

  describe('Confirmations', () => {
    it('should return info confirmation details', async () => {
      const invocation = new RemoteAgentInvocation(mockDefinition, {
        query: 'hi',
      });
      // @ts-expect-error - getConfirmationDetails is protected
      const confirmation = await invocation.getConfirmationDetails(
        new AbortController().signal,
      );

      expect(confirmation).not.toBe(false);
      if (
        confirmation &&
        typeof confirmation === 'object' &&
        confirmation.type === 'info'
      ) {
        expect(confirmation.title).toContain('Test Agent');
        expect(confirmation.prompt).toContain('http://test-agent/card');
      } else {
        throw new Error('Expected confirmation to be of type info');
      }
    });
  });
});
