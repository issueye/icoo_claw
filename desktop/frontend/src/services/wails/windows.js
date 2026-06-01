import { WindowService } from '../../../bindings/icoo_claw/desktop'

export async function openACPMonitorWindow() {
  try {
    await WindowService.OpenACPMonitorWindow()
  } catch {
    window.open('/acp-monitor', 'icoo-claw-acp-monitor', 'width=1120,height=760')
  }
}
