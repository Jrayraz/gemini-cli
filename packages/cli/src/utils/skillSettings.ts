/**
 * @license
 * Copyright 2025 Google LLC
 * SPDX-License-Identifier: Apache-2.0
 */

import {
  SettingScope,
  isLoadableSettingScope,
  type LoadedSettings,
} from '../config/settings.js';

export interface ModifiedScope {
  scope: SettingScope;
  path: string;
}

export type SkillActionStatus = 'success' | 'no-op' | 'error';

export interface SkillActionResult {
  status: SkillActionStatus;
  message: string;
  modifiedScopes: ModifiedScope[];
}

/**
 * Enables a skill by removing it from all writable disabled lists (User and Workspace).
 */
export function enableSkill(
  settings: LoadedSettings,
  skillName: string,
): SkillActionResult {
  const writableScopes = [SettingScope.Workspace, SettingScope.User];
  const foundInScopes: SettingScope[] = [];

  for (const scope of writableScopes) {
    if (isLoadableSettingScope(scope)) {
      const scopeDisabled = settings.forScope(scope).settings.skills?.disabled;
      if (scopeDisabled?.includes(skillName)) {
        foundInScopes.push(scope);
      }
    }
  }

  if (foundInScopes.length === 0) {
    return {
      status: 'no-op',
      modifiedScopes: [],
      message: `Skill "${skillName}" is not disabled.`,
    };
  }

  const modifiedScopes: ModifiedScope[] = [];
  for (const scope of foundInScopes) {
    if (isLoadableSettingScope(scope)) {
      const currentScopeDisabled =
        settings.forScope(scope).settings.skills?.disabled ?? [];
      const newDisabled = currentScopeDisabled.filter(
        (name) => name !== skillName,
      );
      settings.setValue(scope, 'skills.disabled', newDisabled);
      modifiedScopes.push({
        scope,
        path: settings.forScope(scope).path,
      });
    }
  }

  return {
    status: 'success',
    modifiedScopes,
    message: `Skill "${skillName}" enabled by removing it from the disabled list.`,
  };
}

/**
 * Disables a skill by adding it to the disabled list in the specified scope.
 */
export function disableSkill(
  settings: LoadedSettings,
  skillName: string,
  scope: SettingScope,
): SkillActionResult {
  if (!isLoadableSettingScope(scope)) {
    return {
      status: 'error',
      modifiedScopes: [],
      message: `Invalid settings scope: ${scope}`,
    };
  }

  const currentScopeDisabled =
    settings.forScope(scope).settings.skills?.disabled ?? [];

  if (currentScopeDisabled.includes(skillName)) {
    return {
      status: 'no-op',
      modifiedScopes: [],
      message: `Skill "${skillName}" is already disabled in this scope.`,
    };
  }

  // Check if it's disabled in ANY other writable scope to give better feedback
  const otherScope =
    scope === SettingScope.Workspace
      ? SettingScope.User
      : SettingScope.Workspace;
  const isDisabledInOther = settings
    .forScope(otherScope)
    .settings.skills?.disabled?.includes(skillName);

  const newDisabled = [...currentScopeDisabled, skillName];
  settings.setValue(scope, 'skills.disabled', newDisabled);

  const modifiedScopes: ModifiedScope[] = [
    { scope, path: settings.forScope(scope).path },
  ];

  let feedbackText = `Skill "${skillName}" disabled by adding it to the disabled list.`;

  if (isDisabledInOther) {
    feedbackText = `Skill "${skillName}" is now disabled in multiple scopes.`;
  }

  return {
    status: 'success',
    modifiedScopes,
    message: feedbackText,
  };
}
