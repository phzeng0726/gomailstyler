## ADDED Requirements

### Requirement: HTML part 使用 quoted-printable
HTML part SHALL 帶 `Content-Transfer-Encoding: quoted-printable`，內容以 `mime/quotedprintable` 編碼，使每個實體行不超過 76 字元且只含 ASCII。

#### Scenario: 長行與非 ASCII 內容
- **WHEN** `Message` 為一行超過 998 字元、含中文的 HTML
- **THEN** HTML part 原始內容每行不超過 76 字元、只含 ASCII，以 quoted-printable 解碼後精確等於 `Message`

#### Scenario: 一般內容
- **WHEN** `Message` 為 `<p>Hello</p>`
- **THEN** HTML part header 為 `Content-Type: text/html; charset="UTF-8"` 與 `Content-Transfer-Encoding: quoted-printable`，解碼後等於 `<p>Hello</p>`
