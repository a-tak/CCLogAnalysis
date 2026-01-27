/**
 * Date and number formatting utilities
 *
 * これらの関数は複数のページコンポーネントで使用される共通のフォーマット処理を提供します。
 */

/**
 * Validates if a string is in YYYY-MM-DD format
 *
 * @param dateStr - Date string to validate
 * @returns true if valid YYYY-MM-DD format, false otherwise
 */
export function isValidDateFormat(dateStr: string): boolean {
  const datePattern = /^\d{4}-\d{2}-\d{2}$/
  if (!datePattern.test(dateStr)) {
    return false
  }
  const date = new Date(dateStr)
  return !isNaN(date.getTime())
}

/**
 * Formats a date string to Japanese locale format
 *
 * @param dateStr - ISO date string
 * @returns Formatted date string (e.g., "2026年1月27日")
 */
export function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleDateString('ja-JP', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

/**
 * Formats a number with Japanese locale
 *
 * @param num - Number to format
 * @returns Formatted number string with comma separators
 */
export function formatNumber(num: number): string {
  return num.toLocaleString('ja-JP')
}

/**
 * Formats a decimal as percentage
 *
 * @param num - Decimal number (e.g., 0.25 for 25%)
 * @returns Formatted percentage string (e.g., "25.0%")
 */
export function formatPercent(num: number): string {
  return (num * 100).toFixed(1) + '%'
}
