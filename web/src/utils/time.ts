import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'

dayjs.locale('zh-cn')

export function formatDateTime(value?: string | null): string {
  if (!value) {
    return '-'
  }
  return dayjs(value).format('YYYY-MM-DD HH:mm')
}
