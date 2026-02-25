export interface LoginRequest {
  code: string;
  redirect_uri: string;
}

export interface TokenResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface RefreshRequest {
  refresh_token: string;
}
