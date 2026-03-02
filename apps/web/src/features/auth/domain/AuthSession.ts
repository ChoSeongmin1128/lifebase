export interface AuthUrlResponse {
  url: string;
  state: string;
}

export interface AuthCallbackInput {
  code: string;
  state?: string;
  app: "web";
}

export interface AuthTokenPair {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}
