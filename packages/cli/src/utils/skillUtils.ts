/**
 * @license
 * Copyright 2025 Google LLC
 * SPDX-License-Identifier: Apache-2.0
 */

import { SettingScope } from '../config/settings.js';
import type { SkillActionResult } from './skillSettings.js';

/**
 * Formats the result of a skill action for display to the user in the terminal (plain text).
 */
export function formatSkillActionFeedbackPlain(
  result: SkillActionResult,
  skillName: string,
  options: { includeRestartMessage?: boolean } = {},
): string {
  if (result.status === 'no-op' || result.modifiedScopes.length === 0) {
    return result.message;
  }

  const isEnable = result.message.includes('enabled');
  const actionText = isEnable
    ? 'enabled by removing it from the disabled list in'
    : 'disabled by adding it to the disabled list in';

  let feedbackText = '';

  if (result.modifiedScopes.length === 2) {
    const s1 = result.modifiedScopes[0];
    const s2 = result.modifiedScopes[1];
    const label1 =
      s1.scope === SettingScope.Workspace ? 'project' : s1.scope.toLowerCase();
    const label2 =
      s2.scope === SettingScope.Workspace ? 'project' : s2.scope.toLowerCase();

    if (isEnable) {
      feedbackText = `Skill "${skillName}" ${actionText} ${label1} (${s1.path}) and ${label2} (${s2.path}) settings.`;
    } else {
      feedbackText = `Skill "${skillName}" is now disabled in both ${label1} (${s1.path}) and ${label2} (${s2.path}) settings.`;
    }
  } else {
    const s = result.modifiedScopes[0];
    const label =
      s.scope === SettingScope.Workspace ? 'project' : s.scope.toLowerCase();
    feedbackText = `Skill "${skillName}" ${actionText} ${label} settings (${s.path}).`;
  }

  if (options.includeRestartMessage) {
    feedbackText += ' Restart required to take effect.';
  }

  return feedbackText;
}
