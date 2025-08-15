import type { APIResponse, MatchConfig, MatchDetails, User } from '../types/index.ts';

// Configure API base URL - can be updated in production env
const API_BASE_URL = 'http://localhost:8081';

// Helper function to build complete URL
const buildUrl = (endpoint: string): string => `${API_BASE_URL}${endpoint}`;

// Helper function to handle API responses
async function handleResponse<T>(response: Response): Promise<APIResponse<T>> {
  if (response.status === 204) {
    return {}; // No content
  }

  // Try to parse response as JSON, fall back to text if it fails
  let data: unknown;
  const contentType = response.headers.get('content-type');
  
  if (contentType?.includes('application/json')) {
    data = await response.json();
  } else {
    data = await response.text();
  }

  if (!response.ok) {
    let errorMsg = `Error: ${response.status}`;
    if (typeof data === 'string') {
      errorMsg = data;
    } else if (data && typeof data === 'object') {
      const obj = data as Record<string, unknown>;
      errorMsg = (typeof obj.error === 'string' ? obj.error : undefined) ||
                 (typeof obj.message === 'string' ? obj.message : undefined) ||
                 errorMsg;
    }
    return { error: errorMsg };
  }

  return { data };
}

// Helper function for API requests with authorization
async function fetchWithAuth<T>(
  url: string, 
  options: RequestInit = {},
): Promise<APIResponse<T>> {
  // Get token from localStorage
  const token = localStorage.getItem('token');
  
  // Set Authorization header if token exists
  const headers = {
    'Content-Type': 'application/json',
    ...(token && { 'Authorization': `Bearer ${token}` }),
    ...options.headers,
  };
  
  try {
    const response = await fetch(url, {
      ...options,
      headers,
    });
    return handleResponse<T>(response);
  } catch (error) {
    return {
      error: (error as Error).message || 'Network error'
    };
  }
}

// AUTH API CALLS

export async function login(credentials: { handle: string; password: string }): Promise<APIResponse<{ token: string }>> {
  return fetchWithAuth<{ token: string }>(
    buildUrl('/v1/auth/login'),
    {
      method: 'POST',
      body: JSON.stringify(credentials),
    }
  );
}

export async function register(user: { handle: string; password: string; email: string }): Promise<APIResponse<void>> {
  return fetchWithAuth<void>(
    buildUrl('/v1/auth/register'),
    {
      method: 'POST',
      body: JSON.stringify(user),
    }
  );
}

// MATCHMAKING API CALLS

export async function findMatch(config: MatchConfig): Promise<APIResponse<MatchDetails>> {
  return fetchWithAuth<MatchDetails>(
    buildUrl('/v1/match/quick'),
    {
      method: 'POST',
      body: JSON.stringify(config),
    }
  );
}

// GAME API CALLS

export async function submitMove(
  matchId: string, 
  move: string, 
  signature: string
): Promise<APIResponse<unknown>> {
  return fetchWithAuth(
    buildUrl(`/v1/match/${matchId}/append`),
    {
      method: 'POST',
      body: JSON.stringify({
        move,
        signature
      }),
    }
  );
}

export async function resumeMatch(matchId: string): Promise<APIResponse<unknown>> {
  return fetchWithAuth(
    buildUrl(`/v1/match/${matchId}/resume`),
    {
      method: 'POST',
    }
  );
}

// LEADERBOARD API CALLS

export async function getLeaderboard(): Promise<APIResponse<User[]>> {
  return fetchWithAuth<User[]>(
    buildUrl('/v1/leaderboard'),
    {
      method: 'GET',
    }
  );
}

// UTILITIES

export async function generateHMAC(key: string, data: string): Promise<string> {
  const encoder = new TextEncoder();
  const keyData = encoder.encode(key);
  const alg = { name: "HMAC", hash: "SHA-256" };
  
  // Import the key
  const cryptoKey = await window.crypto.subtle.importKey(
    "raw", 
    keyData, 
    alg, 
    false, 
    ["sign"]
  );
  
  // Sign the data
  const signatureBuffer = await window.crypto.subtle.sign(
    "HMAC",
    cryptoKey,
    encoder.encode(data)
  );
  
  // Convert to base64
  return btoa(String.fromCharCode(...new Uint8Array(signatureBuffer)));
}
