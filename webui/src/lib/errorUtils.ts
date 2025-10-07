/**
 * Parses error response from API calls and returns a formatted error message
 */
export async function parseErrorResponse(response: Response): Promise<string> {
  let errorMessage = `HTTP ${response.status}`

  try {
    const errorText = await response.text()
    if (errorText) {
      errorMessage += `: ${errorText}`
    }
  } catch {
    // If we can't read the error, just use status
  }

  return errorMessage
}

/**
 * Handles common API call errors and throws appropriate Error objects
 */
export async function handleApiError(response: Response): Promise<void> {
  // Handle authentication errors
  if (response.status === 401) {
    throw new Error('Authentication required')
  }

  if (!response.ok) {
    // Clone the response before reading to avoid consuming the body stream
    const errorMessage = await parseErrorResponse(response.clone())
    throw new Error(errorMessage)
  }
}