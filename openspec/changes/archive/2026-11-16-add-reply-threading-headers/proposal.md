## Why

peacock-mail-backend 會先寄「維護通知」，之後再寄「維護完成通知」，希望第二封在 Outlook/Exchange 與 Gmail 中顯示為第一封的回覆、歸在同一串。這需要第一封帶自訂的 `Message-ID`、第二封帶 `In-Reply-To` / `References`，但目前 `MailMessage` 無法設定任何 header，`buildHTMLMessage` 也把 header 寫死。

此外，上一個變更（add-unit-tests-and-upgrade-deps）以 `KNOWN ISSUE` 測試記錄了 6 個郵件組裝缺陷。其中 CR/LF 注入一旦開放自訂 header 會變得更嚴重，Subject 未編碼會讓「RE: 中文主旨」在轉寄中被改壞而無法歸串。因此一併修正全部 6 個缺陷。

## What Changes

- `MailMessage` 新增 threading 專用欄位：`MessageID`、`InReplyTo`、`References []string`。函式庫負責角括號格式、格式驗證，以及 `References` 未提供時以 `InReplyTo` 補上
- `MailMessage` 新增 `Headers map[string]string`，用於 `X-*`、`Thread-Topic` 等其他 header；設定到內建 header 時回傳錯誤
- 新增匯出的 sentinel error：`ErrInvalidHeader`、`ErrReservedHeader`、`ErrInvalidMessageID`，呼叫端可用 `errors.Is` 判斷
- 所有 header 值（含 To、Cc、Subject、附件檔名、CID）拒絕 CR/LF，防止 header 注入
- header 以固定順序輸出，自訂 header 依名稱排序
- `Subject` 與非 ASCII 的自訂 header 值使用 RFC 2047 編碼
- 預設送出 `From`（smtpSender）與 `Date`
- HTML part 改用 quoted-printable 編碼並標示 `Content-Transfer-Encoding`
- `SendMail` 不再透過 `append` 修改呼叫端的 `To` slice
- 驗證失敗時 `SendMail` 在連線 SMTP 前就回傳錯誤
- **BREAKING**（行為）：先前會被接受的輸入（header 值含 CR/LF）現在回傳錯誤；寄出的原始訊息格式改變（新增 From/Date、Subject 編碼、HTML 改為 quoted-printable）。公開 API 只有新增，沒有移除或改簽名

## Capabilities

### New Capabilities

- `mail-headers`：threading 專用欄位、自訂 Headers、內建 header 保留清單、header 驗證與 sentinel error、固定輸出順序、RFC 2047 編碼、預設 From/Date
- `message-body-encoding`：HTML part 的 quoted-printable 編碼
- `send-mail-safety`：SendMail 不修改呼叫端資料、驗證先於連線

### Modified Capabilities

- `unit-test-coverage`：「buildHTMLMessage MIME 結構測試」改為驗證新的 header 集合與順序、HTML part 的 quoted-printable，移除「記錄既有缺陷」情境；「SendMail 寄送測試」新增不修改呼叫端 slice、驗證失敗不連線的情境
- `testing-guidelines`：錯誤身份驗證原則改為優先使用本專案新增的 sentinel error

## Impact

- **程式碼**：`message.go`（新欄位）、`manager.go`（header 組裝、驗證、編碼、SendMail）、新增 `errors.go`（sentinel error）、可能新增 `header.go`（驗證與格式化）
- **測試**：`manager_test.go` 的 6 個 `KNOWN ISSUE` 測試改寫為正確行為的測試；新增 header 相關測試
- **文件**：README（新欄位、reply threading 範例、錯誤說明）、`TESTING_GUIDELINES.md`（sentinel error）、`CLAUDE.md`（移除 Known Issues 清單、更新 MIME 說明）
- **相依**：無新增（只用標準庫 `mime`、`mime/quotedprintable`、`net/mail`）
- **下游**：peacock-mail-backend 可升級後設定 MessageID / InReplyTo 完成 threading
- **發版**：建議發佈下一個 minor 版本
