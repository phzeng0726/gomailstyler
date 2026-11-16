## ADDED Requirements

### Requirement: 不修改呼叫端資料
`SendMail` SHALL 不修改傳入 `MailMessage` 中的任何 slice 或 map，包含 `To` 的底層陣列。

#### Scenario: To 有剩餘容量
- **WHEN** 呼叫端的 `To` 長度為 1、容量為 2，且 `Cc` 有一個地址
- **THEN** `SendMail` 之後底層陣列的第 2 個元素維持原值；RCPT 依序為 To 再 Cc

### Requirement: 驗證先於連線
`SendMail` SHALL 先完成訊息組裝與所有驗證，驗證失敗時 MUST 直接回傳錯誤而不連線 SMTP server。

#### Scenario: 無效 header
- **WHEN** `MailMessage` 的 `Headers` 含保留名稱，且 SMTP server 可連線
- **THEN** `SendMail` 回傳 `ErrReservedHeader`，SMTP server 沒有收到任何連線
