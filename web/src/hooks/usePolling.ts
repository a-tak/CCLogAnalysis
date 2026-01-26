import { useEffect } from 'react'

/**
 * 指定された関数を定期的に実行するカスタムフック
 *
 * @param callback - 実行する関数
 * @param interval - ポーリング間隔（ミリ秒）
 * @param enabled - ポーリングを有効にするかどうか（デフォルト: true）
 */
export function usePolling(
  callback: () => void | Promise<void>,
  interval: number,
  enabled: boolean = true
): void {
  useEffect(() => {
    if (!enabled) {
      return
    }

    // 初回実行
    const executeCallback = async () => {
      try {
        await callback()
      } catch (error) {
        // エラーが発生してもポーリングは継続
        console.error('Polling callback error:', error)
      }
    }

    // 即座に実行
    executeCallback()

    // 定期実行
    const intervalId = window.setInterval(executeCallback, interval)

    // クリーンアップ
    return () => {
      clearInterval(intervalId)
    }
  }, [callback, interval, enabled])
}
