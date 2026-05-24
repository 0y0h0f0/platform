import { Switch, Typography } from 'antd'

const { Text } = Typography

interface ArchiveToggleProps {
  checked: boolean
  disabled?: boolean
  onChange: (checked: boolean) => void
}

export function ArchiveToggle({ checked, disabled = false, onChange }: ArchiveToggleProps) {
  return (
    <label className="archive-toggle">
      <Switch
        aria-label="显示归档项目"
        checked={checked}
        disabled={disabled}
        onChange={onChange}
        size="small"
      />
      <Text>显示归档项目</Text>
    </label>
  )
}
