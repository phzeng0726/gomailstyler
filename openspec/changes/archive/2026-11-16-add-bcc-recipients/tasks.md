## 1. 實作

- [x] 1.1 `message.go` 新增 `Bcc []string` 與註解
- [x] 1.2 `header.go`：`buildHeaders` 驗證 `Bcc[i]` 不含 CR/LF；`To` 為空時不輸出 `To` header
- [x] 1.3 `manager.go`：`SendMail` 收件人清單依序加入 To、Cc、Bcc

## 2. 測試

- [x] 2.1 `header_test.go`：`Bcc[1]` 含 CR/LF 回傳 `ErrInvalidHeader` 與精確錯誤字串；buildHeaders 輸出不含 Bcc；To 為空時 header 名稱順序為 From、Subject、Date、MIME-Version、Content-Type
- [x] 2.2 `manager_test.go`：To+Cc+Bcc 的 RCPT 順序；DATA 不含 Bcc 地址也沒有 `Bcc` header；只有 Bcc 的寄送；Bcc 驗證失敗不連線；不修改呼叫端 To/Cc slice
- [x] 2.3 `go vet ./...`、`go test -cover ./...` 通過，根 package 維持 100%

## 3. 範例與文件

- [x] 3.1 新增 `examples/02-html-content/04-with-bcc/main.go`：示範 To + Bcc 與只有 Bcc 兩封信
- [x] 3.2 `examples/README.md`：目錄樹、範例說明、選擇表加入 04-with-bcc
- [x] 3.3 README：MailMessage 新增 `Bcc`、Bcc 使用說明（只進 envelope、只有 Bcc 的垃圾信風險）、範例清單、升級注意事項補充「To 為空時不再輸出 To header」
- [x] 3.4 `CLAUDE.md`：MIME Message Construction 補充 Bcc 只進 envelope、To 為空時省略
