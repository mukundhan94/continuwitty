import { ApiError } from '../api/http'

export function describeError(error: unknown): string {
  if (error instanceof ApiError) {
    return error.detail
  }
  if (error instanceof Error) {
    return error.message
  }
  return 'Unexpected error'
}
