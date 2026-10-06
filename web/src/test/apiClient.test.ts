import { describe, it, expect, beforeEach } from 'vitest';
import {
  getAccessToken,
  setAccessToken,
  getRefreshToken,
  setRefreshToken,
} from '../api/client';

describe('Token Storage & Management', () => {
  beforeEach(() => {
    setAccessToken(null);
    setRefreshToken(null);
    localStorage.clear();
  });

  it('stores access token in memory only', () => {
    expect(getAccessToken()).toBeNull();
    setAccessToken('access-token-123');
    expect(getAccessToken()).toBe('access-token-123');
    expect(localStorage.getItem('planly_access_token')).toBeNull();
  });

  it('stores refresh token in localStorage', () => {
    expect(getRefreshToken()).toBeNull();
    setRefreshToken('refresh-token-xyz');
    expect(getRefreshToken()).toBe('refresh-token-xyz');
    expect(localStorage.getItem('planly_refresh_token')).toBe('refresh-token-xyz');

    setRefreshToken(null);
    expect(getRefreshToken()).toBeNull();
    expect(localStorage.getItem('planly_refresh_token')).toBeNull();
  });
});
