export interface ErrorBody {
  code: string;
  message: string;
}

export interface ErrorResponse {
  error: ErrorBody;
}

export interface PaginatedResponse<T> {
  data: T[];
  next_cursor?: string;
}

export interface HealthResponse {
  status: string;
  time: string;
}
