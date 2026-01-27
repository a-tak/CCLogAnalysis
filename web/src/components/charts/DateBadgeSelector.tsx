import { CardContent } from '@/components/ui/card'

interface DateBadgeSelectorProps {
  timeSeriesData: Array<{ periodStart: string | Date }>
  selectedDate: string | null
  onDateClick: (dateStr: string) => void
  period: string
  loading: boolean
}

export function DateBadgeSelector({
  timeSeriesData,
  selectedDate,
  onDateClick,
  period,
  loading,
}: DateBadgeSelectorProps) {
  if (period !== 'day' || loading || !timeSeriesData || timeSeriesData.length === 0) {
    return null
  }

  return (
    <CardContent className="pt-0">
      <div className="text-sm text-muted-foreground mb-2">日付を選択してドリルダウン:</div>
      <div className="flex flex-wrap gap-2">
        {[...timeSeriesData].reverse().map((item) => {
          const date = new Date(item.periodStart)
          const dateStr = date.toISOString().split('T')[0]
          const displayDate = `${date.getMonth() + 1}/${date.getDate()}`
          const isSelected = selectedDate === dateStr
          return (
            <button
              key={dateStr}
              type="button"
              onClick={() => onDateClick(dateStr)}
              className={
                isSelected
                  ? 'inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 border-transparent bg-primary text-primary-foreground hover:bg-primary/80 cursor-pointer'
                  : 'inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 border-border text-foreground cursor-pointer hover:bg-primary hover:text-primary-foreground'
              }
            >
              {displayDate}
            </button>
          )
        })}
      </div>
    </CardContent>
  )
}
