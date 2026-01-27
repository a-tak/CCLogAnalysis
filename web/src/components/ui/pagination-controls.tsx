import { Button } from '@/components/ui/button'
import { ChevronLeft, ChevronRight } from 'lucide-react'

interface PaginationControlsProps {
  currentPage: number
  totalPages: number
  onPageChange: (page: number) => void
  startIndex?: number
  endIndex?: number
  totalItems?: number
}

export function PaginationControls({
  currentPage,
  totalPages,
  onPageChange,
  startIndex,
  endIndex,
  totalItems,
}: PaginationControlsProps) {
  if (totalPages <= 1) {
    return null
  }

  const showItemCount = startIndex !== undefined && endIndex !== undefined && totalItems !== undefined

  return (
    <div className="flex items-center justify-between">
      {showItemCount ? (
        <p className="text-sm text-muted-foreground">
          {startIndex + 1} - {Math.min(endIndex, totalItems)} / {totalItems}件
        </p>
      ) : (
        <div />
      )}
      <div className="flex gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={() => onPageChange(currentPage - 1)}
          disabled={currentPage === 1}
        >
          <ChevronLeft className="h-4 w-4 mr-1" />
          前へ
        </Button>
        <span className="flex items-center px-4 text-sm text-muted-foreground">
          {currentPage} / {totalPages}
        </span>
        <Button
          variant="outline"
          size="sm"
          onClick={() => onPageChange(currentPage + 1)}
          disabled={currentPage === totalPages}
        >
          次へ
          <ChevronRight className="h-4 w-4 ml-1" />
        </Button>
      </div>
    </div>
  )
}
