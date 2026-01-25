import { Link, Outlet } from 'react-router-dom'
import { ScanStatusIndicator } from '@/components/scan/ScanStatusIndicator'

export function Layout() {
  return (
    <>
      <div className="min-h-screen bg-background">
        <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
          <div className="container flex h-14 max-w-screen-2xl items-center">
            <div className="mr-4 flex">
              <Link to="/" className="mr-6 flex items-center space-x-2">
                <span className="font-bold">Claude Code Log Analysis</span>
              </Link>
              <nav className="flex items-center gap-6 text-sm">
                <Link
                  to="/"
                  className="transition-colors hover:text-foreground/80 text-foreground/60"
                >
                  Projects
                </Link>
              </nav>
            </div>
          </div>
        </header>
        <main className="container mx-auto py-6">
          <Outlet />
        </main>
      </div>

      {/* スキャン状態インジケーター */}
      <ScanStatusIndicator />
    </>
  )
}
