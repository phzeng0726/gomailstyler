## Why

`MailMessage` 只有 `To` 與 `Cc`，無法加入「收得到信、但其他收件人看不到」的收件人。寄送通知時常需要暗中副本給管理者或稽核信箱，或一次寄給多位彼此不該看到對方地址的使用者，目前只能拆成多封信。

## What Changes

- `MailMessage` 新增 `Bcc []string`
- Bcc 地址只加入 SMTP envelope 收件人（RCPT），順序為 To → Cc → Bcc；**不**寫入任何 header
- Bcc 地址沿用 CR/LF 檢查，違反時回傳 `ErrInvalidHeader`（`Bcc[i] contains CR or LF`），在連線前就失敗
- `To` 為空時不輸出 `To` header（RFC 5322 允許），讓「只有 Bcc」的寄送不會產生空的 `To:` 行
- 新增範例 `examples/02-html-content/04-with-bcc/`，並更新 README 與 `examples/README.md`

## Capabilities

### New Capabilities

- `bcc-recipients`：Bcc 欄位、envelope 收件人順序、Bcc 不出現在訊息內容、To 為空時省略 To header

### Modified Capabilities

- `mail-headers`：「拒絕 CR/LF」需求的欄位清單加入 `Bcc`

## Impact

- **程式碼**：`message.go`（新欄位）、`header.go`（Bcc 驗證、To 為空時省略）、`manager.go`（收件人清單加入 Bcc）
- **測試**：`header_test.go`、`manager_test.go`（假 SMTP server 驗證 RCPT 與 DATA 內容）
- **文件**：README（MailMessage 欄位、Bcc 說明、範例清單）、`examples/README.md`、`CLAUDE.md`
- **範例**：新增 `04-with-bcc`
- **相容性**：只新增欄位；`To` 為空時的輸出從 `To: `（空值）改為不輸出，收件端顯示不受影響
