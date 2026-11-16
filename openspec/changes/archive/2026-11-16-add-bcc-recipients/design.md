## Context

- `SendMail` 目前以新 slice 組合 `To` + `Cc` 作為 envelope 收件人；`buildHeaders`（header.go）一律輸出 `To` header，即使 `To` 為空（產生 `To: `）。
- header 已有完整驗證與固定順序機制，`Bcc` 已在保留名稱清單中，無法透過 `Headers` 設定。
- 使用者要求每個新功能都附可執行的範例。

## Goals / Non-Goals

**Goals:**
- 支援 Bcc 收件人，且 Bcc 地址絕不出現在訊息內容
- 支援只有 Bcc 的寄送
- 提供可直接執行的範例

**Non-Goals:**
- 不對 To/Cc/Bcc 去除重複地址（重複時 SMTP server 可能寄出多份，由呼叫端負責）
- 不新增「沒有任何收件人」的驗證；維持現狀由 SMTP 回報錯誤
- 不輸出 `To: undisclosed-recipients:;` 之類的群組語法
- 不支援每位 Bcc 收件人各自收到含 `Bcc` header 的副本（RFC 5322 3.6.3 的另一種做法）

## Decisions

### D1. Bcc 只進 envelope，不寫 header
RFC 5322 允許三種 Bcc 處理方式，最常見也最安全的是「移除 Bcc 行、只透過 envelope 投遞」。`buildHeaders` 完全不讀 `mm.Bcc` 的值（只做驗證），由 `SendMail` 加入收件人清單。
- 替代方案：寫入 `Bcc` header 再期待 SMTP server 移除——依賴伺服器行為，可能洩漏地址。

### D2. 驗證放在 buildHeaders
雖然 Bcc 不輸出 header，仍在 `buildHeaders` 中以 `validateHeaderValue("Bcc[i]", addr)` 檢查，與 To/Cc 一致，確保 `SendMail` 在連線前就回傳 `ErrInvalidHeader`。`net/smtp` 對 RCPT 也會拒絕 CR/LF，但錯誤型別不同且已建立連線。

### D3. To 為空時省略 To header
在 `buildHeaders` 中與 Cc 相同處理：`len(mm.To) > 0` 才輸出。RFC 5322 中 `To` 為選填，空的 `To:` 行反而不合 address-list 語法。

### D4. 收件人清單
```go
allRecipients := make([]string, 0, len(mm.To)+len(mm.Cc)+len(mm.Bcc))
allRecipients = append(allRecipients, mm.To...)
allRecipients = append(allRecipients, mm.Cc...)
allRecipients = append(allRecipients, mm.Bcc...)
```

### D5. 範例 04-with-bcc
寄兩封信，`MAIL_RECEIVER` 都放在 Bcc：
1. To = `SMTP_SENDER`（寄給自己）、Bcc = `MAIL_RECEIVER`：收件人應看到 To 是寄件者、看不到自己的地址
2. 只有 Bcc = `MAIL_RECEIVER`：收件人應看到沒有 To（郵件軟體可能顯示為空或「未公開的收件者」）

## Risks / Trade-offs

- [只有 Bcc 時，部分郵件軟體或垃圾信過濾器會對沒有 To 的信提高可疑分數] → README 建議一般情況把寄件者或固定信箱放在 To，Bcc 給實際收件人
- [`To` 為空時輸出改變（從 `To: ` 變為不輸出）] → 先前的空 `To:` 本來就不合語法，收件端顯示不受影響；README 升級注意事項補充說明
- [Bcc 收件人在伺服器記錄（Received header 的 `for <addr>`）中可能出現] → 屬於 SMTP server 行為，函式庫無法控制；只有單一收件人時部分伺服器會寫入 `for`，README 說明
