## 1. 基礎型別與錯誤

- [x] 1.1 新增 `errors.go`：`ErrInvalidHeader`、`ErrReservedHeader`、`ErrInvalidMessageID`
- [x] 1.2 `message.go` 新增 `MessageID`、`InReplyTo`、`References`、`Headers` 欄位與註解
- [x] 1.3 `Manager` 新增 `now func() time.Time`，`NewManager` 預設 `time.Now`；`validate()` 檢查 smtpSender 不含 CR/LF（回傳 `ErrInvalidHeader`）

## 2. Header 組裝（header.go）

- [x] 2.1 實作 `validateHeaderValue`、`validateHeaderName`、`isReservedHeader`
- [x] 2.2 實作 `normalizeMessageID`（去角括號、格式驗證、加回角括號）
- [x] 2.3 實作 References 格式化：`InReplyTo` 補位、空白分隔、超過 78 字元折行
- [x] 2.4 實作 `buildHeaders`：固定順序、From/Date、Subject 與自訂值 RFC 2047 編碼、自訂 header 排序與重複偵測、To/Cc/附件檔名/CID 驗證
- [x] 2.5 新增 `header_test.go`：上述每個函式的成功、邊界與錯誤情境（`errors.Is` + 精確錯誤字串）

## 3. 訊息組裝與寄送

- [x] 3.1 `buildHTMLMessage` 改為 `([]byte, error)`，改用 `buildHeaders` 的有序結果輸出
- [x] 3.2 HTML part 改為 quoted-printable 並加上 `Content-Transfer-Encoding`
- [x] 3.3 `SendMail`：先組裝再連線；收件者清單改用新 slice
- [x] 3.4 `smtp_server_test.go` 新增連線計數

## 4. 更新既有測試

- [x] 4.1 `manager_test.go`：最小訊息改為驗證新的 header 集合與順序（注入固定時間）、HTML part 改為 quoted-printable 解碼比對
- [x] 4.2 將 6 個 `KNOWN ISSUE` 測試改寫為正確行為：CR/LF 回傳 `ErrInvalidHeader`、中文主旨編碼、順序固定、From/Date 存在、To slice 未被改寫、HTML 長行 QP 後每行 ≤ 76
- [x] 4.3 新增「完整 threading 訊息」精確 header 區塊比對（boundary 以正規表示式固定）
- [x] 4.4 新增 `SendMail` 驗證失敗不連線測試（連線計數為 0）、帶 threading 欄位的寄送測試
- [x] 4.5 `go test -cover ./...` 兩個 package 維持 ≥ 85%（根 package 目標 100%）、`go vet ./...` 通過

## 5. 文件

- [x] 5.1 README：MailMessage 新欄位、Reply threading 範例（含 Gmail/Outlook 主旨一致的注意事項）、sentinel error 說明、行為變更說明
- [x] 5.2 `TESTING_GUIDELINES.md`：錯誤身份驗證改為優先使用 sentinel error；header 區塊可精確比對
- [x] 5.3 `CLAUDE.md`：移除 Known Issues 清單，更新 MIME Message Construction 說明（header 順序、驗證、編碼、QP）
- [x] 5.5 新增 `examples/02-html-content/03-reply-thread/main.go`：寄出「維護通知」（自訂 MessageID）後，再寄「維護完成通知」（InReplyTo + `RE: ` 主旨 + 自訂 Headers），並更新 README 與 `examples/README.md` 的範例清單
- [x] 5.4 README 新增「升級注意事項（Upgrade Notes）」：不帶欄位名稱的 `MailMessage{...}` 會編譯失敗、含 CR/LF 的輸入改為回傳錯誤、新增 From/Date 與編碼變更（收件端顯示不變）、From 固定為 smtpSender（依賴 relay 改寫 From 顯示名稱者需注意）
