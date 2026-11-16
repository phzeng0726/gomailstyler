## ADDED Requirements

### Requirement: Bcc 收件人
`MailMessage.Bcc` 中的地址 SHALL 加入 SMTP envelope 收件人，`RCPT TO` 的順序 MUST 為 To、Cc、Bcc（各自保持輸入順序）。

#### Scenario: To、Cc、Bcc 都有
- **WHEN** `To` 為 `[a@example.com]`、`Cc` 為 `[c@example.com]`、`Bcc` 為 `[x@example.com, y@example.com]`
- **THEN** SMTP server 依序收到 RCPT `a@example.com`、`c@example.com`、`x@example.com`、`y@example.com`

#### Scenario: 不修改呼叫端資料
- **WHEN** 呼叫端的 `To` 與 `Cc` 有剩餘容量且 `Bcc` 非空
- **THEN** `SendMail` 後呼叫端的底層陣列未被改寫

### Requirement: Bcc 不出現在訊息內容
Bcc 地址 SHALL 不出現在寄出的訊息中：不輸出 `Bcc` header，且原始訊息（header 與 body）MUST 不包含任何 Bcc 地址字串。

#### Scenario: 原始訊息不含 Bcc
- **WHEN** 寄出含 `Bcc: [secret@example.com]` 的訊息
- **THEN** SMTP server 收到的 DATA 不含 `secret@example.com`，也沒有 `Bcc` header

### Requirement: To 為空時省略 To header
`To` 為空時 SHALL 不輸出 `To` header；`To` 有值時行為不變。

#### Scenario: 只有 Bcc
- **WHEN** `To` 與 `Cc` 皆為空、`Bcc` 為 `[x@example.com]`
- **THEN** 訊息不含 `To` 與 `Cc` header，header 順序為 `From`、`Subject`、`Date`、`MIME-Version`、`Content-Type`；RCPT 只有 `x@example.com`

### Requirement: Bcc 範例
專案 SHALL 提供 `examples/02-html-content/04-with-bcc/main.go`，可用 `examples/.env` 直接執行，示範「To + Bcc」與「只有 Bcc」兩種寄法。

#### Scenario: 執行範例
- **WHEN** 開發者在範例目錄執行 `go run main.go`
- **THEN** `MAIL_RECEIVER` 以 Bcc 收到兩封信，且信中看不到自己的地址出現在 To/Cc
