/**
 * Common form validation helpers for template-driven forms.
 * Used with Angular's built-in required/minlength/pattern validators.
 */

/** Check if a value looks like a valid E.164 phone number */
export function isValidPhone(phone: string): boolean {
  return /^\+\d{10,15}$/.test(phone.trim());
}

/** Check minimum password strength */
export function isValidPassword(pw: string): boolean {
  return pw.length >= 8;
}

/** Check if a value is a valid UUID v4 */
export function isValidUUID(val: string): boolean {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(val);
}

/** Get a human-readable error message for a form control error map */
export function getErrorMessage(errors: Record<string, unknown> | null, label = 'This field'): string {
  if (!errors) return '';
  if (errors['required']) return `${label} is required`;
  if (errors['minlength']) {
    const e = errors['minlength'] as { requiredLength: number };
    return `${label} must be at least ${e.requiredLength} characters`;
  }
  if (errors['maxlength']) {
    const e = errors['maxlength'] as { requiredLength: number };
    return `${label} cannot exceed ${e.requiredLength} characters`;
  }
  if (errors['email']) return `Please enter a valid email address`;
  if (errors['pattern']) return `${label} has an invalid format`;
  return `${label} is invalid`;
}
