import { describe, it, expect } from 'vitest';
import { assertApiTokenConfigured } from './require-api-token';

describe('assertApiTokenConfigured', () => {
  it('passes when auth is not required, token or not', () => {
    expect(() => assertApiTokenConfigured({})).not.toThrow();
    expect(() => assertApiTokenConfigured({ AUTH_REQUIRED: 'false' })).not.toThrow();
  });

  it('passes when required and a token is set', () => {
    expect(() => assertApiTokenConfigured({ AUTH_REQUIRED: 'true', API_TOKEN: 'tok' })).not.toThrow();
  });

  it.each([undefined, '', '   '])('refuses to start when required and the token is %j', (token) => {
    expect(() => assertApiTokenConfigured({ AUTH_REQUIRED: 'true', API_TOKEN: token })).toThrow(/API_TOKEN/);
  });

  it('reads the flag case-insensitively, so a typo-ish "TRUE" still fails closed', () => {
    expect(() => assertApiTokenConfigured({ AUTH_REQUIRED: ' TRUE ' })).toThrow(/API_TOKEN/);
  });
});
