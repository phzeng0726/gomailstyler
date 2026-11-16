## MODIFIED Requirements

### Requirement: 拒絕 CR/LF
`Subject`、`To`、`Cc` 與 `Bcc` 的每個地址、`Headers` 的值、附件 `FileName`、inline 圖片 `CID` 與 `FileName` SHALL 不含 `\r` 或 `\n`；違反時 MUST 回傳包裝 `ErrInvalidHeader` 的錯誤，訊息包含欄位名稱。`NewManager` MUST 以同樣規則驗證 `smtpSender`。

#### Scenario: Subject 注入
- **WHEN** `Subject` 為 `Hello\r\nBcc: evil@example.com`
- **THEN** 回傳錯誤，`errors.Is(err, ErrInvalidHeader)` 為 true，且沒有產生任何訊息

#### Scenario: 收件者或檔名注入
- **WHEN** `To` 某個地址、`Headers` 某個值、附件 `FileName` 或 inline `CID` 含 `\n`
- **THEN** 回傳 `ErrInvalidHeader`，訊息指出對應欄位

#### Scenario: sender 含換行
- **WHEN** `WithSMTP` 的 sender 含 `\r\n`
- **THEN** `NewManager` 回傳錯誤，`errors.Is(err, ErrInvalidHeader)` 為 true

#### Scenario: Bcc 含換行
- **WHEN** `Bcc` 第 2 個地址含 `\r\n`
- **THEN** 回傳錯誤，`errors.Is(err, ErrInvalidHeader)` 為 true，訊息精確為 `invalid header: Bcc[1] contains CR or LF`，且 `SendMail` 不連線 SMTP server
