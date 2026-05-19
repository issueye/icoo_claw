Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$gatewayUrl = "http://127.0.0.1:8080"
$agentId = "agent_desktop_default"
$prompt = "hello from scripted smoke test"
$requestId = "req_smoke_" + [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()

try {
  $null = Invoke-RestMethod -Uri "$gatewayUrl/health" -Method Get -TimeoutSec 3
} catch {
  throw "gateway is not reachable at $gatewayUrl"
}

$conversationBody = @{
  agent_id = $agentId
  title = "Scripted Smoke Test"
} | ConvertTo-Json

$conversation = Invoke-RestMethod `
  -Uri "$gatewayUrl/v1/conversations" `
  -Method Post `
  -ContentType "application/json" `
  -Body $conversationBody `
  -TimeoutSec 5

$ws = [System.Net.WebSockets.ClientWebSocket]::new()
$cts = [System.Threading.CancellationTokenSource]::new()
$cts.CancelAfter(15000)
$uri = [Uri]"ws://127.0.0.1:8080/v1/ws/chat"
$ws.ConnectAsync($uri, $cts.Token).GetAwaiter().GetResult()

$payload = @{
  type = "chat.start"
  conversation_id = $conversation.id
  request_id = $requestId
  prompt = $prompt
} | ConvertTo-Json -Compress

$sendBytes = [System.Text.Encoding]::UTF8.GetBytes($payload)
$segment = [ArraySegment[byte]]::new($sendBytes)
$ws.SendAsync($segment, [System.Net.WebSockets.WebSocketMessageType]::Text, $true, $cts.Token).GetAwaiter().GetResult() | Out-Null

$buffer = New-Object byte[] 4096
$messages = @()

while ($ws.State -eq [System.Net.WebSockets.WebSocketState]::Open) {
  $stream = New-Object System.IO.MemoryStream
  do {
    $recvSegment = [ArraySegment[byte]]::new($buffer)
    $result = $ws.ReceiveAsync($recvSegment, $cts.Token).GetAwaiter().GetResult()
    if ($result.MessageType -eq [System.Net.WebSockets.WebSocketMessageType]::Close) {
      $ws.CloseAsync([System.Net.WebSockets.WebSocketCloseStatus]::NormalClosure, "done", $cts.Token).GetAwaiter().GetResult() | Out-Null
      break
    }
    $stream.Write($buffer, 0, $result.Count)
  } while (-not $result.EndOfMessage)

  if ($stream.Length -eq 0) {
    continue
  }

  $text = [System.Text.Encoding]::UTF8.GetString($stream.ToArray())
  $message = $text | ConvertFrom-Json
  $messages += $message

  if ($message.type -eq "message.completed" -or $message.type -eq "message.error") {
    break
  }
}

$persisted = Invoke-RestMethod -Uri "$gatewayUrl/v1/conversations/$($conversation.id)/messages" -Method Get -TimeoutSec 5

[PSCustomObject]@{
  conversationId = $conversation.id
  websocketEvents = $messages
  persistedMessages = $persisted.messages
} | ConvertTo-Json -Depth 8
