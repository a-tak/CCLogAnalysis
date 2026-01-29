import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import userEvent from '@testing-library/user-event'
import { SessionListTab } from './SessionListTab'
import type { SessionSummary } from '@/lib/api/types'

// テスト用ラッパーコンポーネント
function SessionListTabWithRouter(props: React.ComponentProps<typeof SessionListTab>) {
  return (
    <BrowserRouter>
      <SessionListTab {...props} />
    </BrowserRouter>
  )
}

describe('SessionListTab', () => {
  const mockSessions: SessionSummary[] = [
    {
      id: 'session-1',
      projectName: 'test-project',
      gitBranch: 'main',
      startTime: '2026-01-25T10:00:00Z',
      firstUserMessage: 'Test message 1',
      totalTokens: 1000,
      errorCount: 0,
    },
    {
      id: 'session-2',
      projectName: 'test-project',
      gitBranch: 'develop',
      startTime: '2026-01-28T14:00:00Z',
      firstUserMessage: 'Test message 2',
      totalTokens: 2000,
      errorCount: 1,
    },
    {
      id: 'session-3',
      projectName: 'test-project',
      gitBranch: 'feature',
      startTime: '2026-01-28T16:00:00Z',
      firstUserMessage: 'Test message 3',
      totalTokens: 3000,
      errorCount: 0,
    },
  ]

  describe('日付フィルタ機能', () => {
    it('フィルタが存在しない場合、すべてのセッションを表示する', () => {
      render(<SessionListTabWithRouter sessions={mockSessions} loading={false} />)

      expect(screen.getByText('Test message 1')).toBeInTheDocument()
      expect(screen.getByText('Test message 2')).toBeInTheDocument()
      expect(screen.getByText('Test message 3')).toBeInTheDocument()
    })

    it('開始日を指定した場合、その日付以降のセッションのみを表示する', async () => {
      const user = userEvent.setup()
      render(<SessionListTabWithRouter sessions={mockSessions} loading={false} />)

      // 開始日を2026-01-28に設定
      const dateInputs = screen.getAllByDisplayValue('')
      await user.type(dateInputs[0], '2026-01-28')

      // 2026-01-25のセッションは表示されない
      expect(screen.queryByText('Test message 1')).not.toBeInTheDocument()
      // 2026-01-28以降のセッションは表示される
      expect(screen.getByText('Test message 2')).toBeInTheDocument()
      expect(screen.getByText('Test message 3')).toBeInTheDocument()
    })

    it('終了日を指定した場合、その日付以前のセッションのみを表示する', async () => {
      const user = userEvent.setup()
      render(<SessionListTabWithRouter sessions={mockSessions} loading={false} />)

      // 終了日を2026-01-25に設定
      const dateInputs = screen.getAllByDisplayValue('')
      await user.type(dateInputs[1], '2026-01-25')

      // 2026-01-25以前のセッションのみ表示
      expect(screen.getByText('Test message 1')).toBeInTheDocument()
      // 2026-01-28のセッションは表示されない
      expect(screen.queryByText('Test message 2')).not.toBeInTheDocument()
      expect(screen.queryByText('Test message 3')).not.toBeInTheDocument()
    })

    it('開始日と終了日を両方指定した場合、その範囲内のセッションのみを表示する', async () => {
      const user = userEvent.setup()
      render(<SessionListTabWithRouter sessions={mockSessions} loading={false} />)

      const dateInputs = screen.getAllByDisplayValue('')

      // 2026-01-25から2026-01-25の範囲を指定
      await user.type(dateInputs[0], '2026-01-25')
      await user.type(dateInputs[1], '2026-01-25')

      // 2026-01-25のセッションのみ表示
      expect(screen.getByText('Test message 1')).toBeInTheDocument()
      expect(screen.queryByText('Test message 2')).not.toBeInTheDocument()
      expect(screen.queryByText('Test message 3')).not.toBeInTheDocument()
    })

    it('クリアボタンをクリックするとすべてのセッションが表示される', async () => {
      const user = userEvent.setup()
      render(<SessionListTabWithRouter sessions={mockSessions} loading={false} />)

      const dateInputs = screen.getAllByDisplayValue('')

      // フィルタを設定
      await user.type(dateInputs[0], '2026-01-28')

      // クリアボタンが表示されていることを確認
      const clearButton = screen.getByRole('button', { name: 'クリア' })
      expect(clearButton).toBeInTheDocument()

      // クリアボタンをクリック
      await user.click(clearButton)

      // すべてのセッションが表示される
      expect(screen.getByText('Test message 1')).toBeInTheDocument()
      expect(screen.getByText('Test message 2')).toBeInTheDocument()
      expect(screen.getByText('Test message 3')).toBeInTheDocument()
    })

    it('フィルタ条件に一致するセッションが0件の場合、メッセージを表示する', async () => {
      const user = userEvent.setup()
      render(<SessionListTabWithRouter sessions={mockSessions} loading={false} />)

      const dateInputs = screen.getAllByDisplayValue('')

      // 存在しない日付範囲を指定
      await user.type(dateInputs[1], '2026-01-20')

      // エラーメッセージが表示される
      expect(screen.getByText('条件に一致するセッションが見つかりません')).toBeInTheDocument()
    })
  })

  describe('ローディング状態', () => {
    it('loading が true の場合、ローディングスピナーを表示する', () => {
      const { container } = render(<SessionListTabWithRouter sessions={[]} loading={true} />)

      const spinner = container.querySelector('.animate-spin')
      expect(spinner).toBeInTheDocument()
    })
  })

  describe('空のセッションリスト', () => {
    it('セッションが0件の場合、メッセージを表示する', () => {
      render(<SessionListTabWithRouter sessions={[]} loading={false} />)

      expect(screen.getByText('セッションが見つかりません')).toBeInTheDocument()
    })
  })

  describe('ページネーション', () => {
    // 20件以上のセッションでページネーションをテスト
    const manyMockSessions: SessionSummary[] = Array.from({ length: 25 }, (_, i) => ({
      id: `session-${i}`,
      projectName: 'test-project',
      gitBranch: 'main',
      startTime: new Date(2026, 0, 25 + Math.floor(i / 20)).toISOString(),
      firstUserMessage: `Test message ${i + 1}`,
      totalTokens: (i + 1) * 1000,
      errorCount: i % 3 === 0 ? 1 : 0,
    }))

    it('ページネーション表示が正しい', () => {
      render(<SessionListTabWithRouter sessions={manyMockSessions} loading={false} />)

      // ページネーション表示が存在することを確認
      expect(screen.getByText(/前へ/)).toBeInTheDocument()
      expect(screen.getByText(/次へ/)).toBeInTheDocument()
    })

    it('フィルタ後のページネーションが正しく更新される', async () => {
      const user = userEvent.setup()
      render(<SessionListTabWithRouter sessions={manyMockSessions} loading={false} />)

      const dateInputs = screen.getAllByDisplayValue('')

      // 2026-01-26のセッションのみに絞る
      await user.type(dateInputs[0], '2026-01-26')

      // フィルタ後の件数が表示される（最初は5件）
      expect(screen.getByText(/5件/)).toBeInTheDocument()
    })
  })
})
