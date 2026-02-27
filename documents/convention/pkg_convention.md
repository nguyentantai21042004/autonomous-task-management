# Chuẩn hóa package (pkg) – knowledge-srv

Tài liệu này mô tả convention cho các package trong `pkg/` để code đồng nhất, dễ bảo trì và test. Các pkg **gemini**, **voyage**, **redis**, **qdrant** đã áp dụng chuẩn này.

---

## 1. Cấu trúc file (4 file nền + tùy chọn)

### Bắt buộc – 4 file

| File                       | Nội dung                                                                              |
| -------------------------- | ------------------------------------------------------------------------------------- |
| **interface.go**           | Interface chính + constructor `New(...)` hoặc `NewXxx(...)` trả về interface          |
| **type.go** / **types.go** | Config struct, impl struct (unexported), request/response types                       |
| **constant.go**            | Hằng số: timeout, endpoint, default limit, **string literal** (distance metric, v.v.) |
| **\<pkg\>.go**             | Toàn bộ logic: method của impl (không tách nhiều file trừ khi thật cần)               |

### Tùy chọn – khi cần

| File          | Khi dùng                                                                        |
| ------------- | ------------------------------------------------------------------------------- |
| **errors.go** | Sentinel errors (`var ErrXxx = errors.New(...)`) + helper `WrapError(err, msg)` |
| **util.go**   | Helper dùng chung (convert value, validate config/vector, map string→enum)      |

- **Không** tách collection/points/search thành file riêng nếu chỉ vài trăm dòng; gộp vào `<pkg>.go`.

---

## 2. Interface và implementation

- **Interface**: tên rõ (vd. `IGemini`, `IVoyage`, `IRedis`, `IQdrant`), doc ngắn + “Implementations are safe for concurrent use” nếu đúng.
- **Impl**: struct **unexported** (vd. `geminiImpl`, `voyageImpl`, `redisImpl`, `qdrantImpl`).
- **Constructor**: trả về **interface**, không trả về concrete type.
  - Ví dụ: `New(cfg Config) (IRedis, error)`, `NewGemini(cfg GeminiConfig) IGemini`.

---

## 3. Config và validation

- Mỗi pkg có **config struct** (vd. `GeminiConfig`, `RedisConfig`, `QdrantConfig`).
- Có thể có alias `Config = XxxConfig` để tương thích (vd. config layer gọi `New(cfg)`).
- **Validate** trong constructor hoặc method đầu tiên; fail sớm với message rõ (có prefix `pkg:` nếu muốn).
- **Constants** cho mọi magic string/number:
  - Timeout: `DefaultTimeout`, `DefaultPingTimeout`, `DefaultConnectTimeout`.
  - Limit: `DefaultSearchLimit`.
  - String dùng trong switch/config: `DistanceCosine`, `DistanceEuclidean`, …

---

## 4. Dependency (client) – dùng pointer

- HTTP/gRPC/Redis client lưu dưới dạng **pointer** (vd. `*pkghttp.Client`, `*goredis.Client`) để tránh copy struct và thống nhất với các pkg khác.
- Ví dụ: `httpClient *pkghttp.Client`, không dùng `httpClient pkghttp.Client` rồi gán `*pkghttp.NewClient(...)`.

---

## 5. Error và doc

- Lỗi: wrap bằng `fmt.Errorf("...: %w", err)`; message rõ, có thể dùng `WrapError(err, msg)` trong errors.go.
- Doc comment ngắn cho: interface, constructor, method public chính.

---

## 6. Checklist khi tạo/sửa pkg

- [ ] Đủ 4 file: interface.go, type.go (hoặc types.go), constant.go, \<pkg\>.go.
- [ ] Interface + impl unexported; constructor trả về interface.
- [ ] Config struct + validate; constants cho magic string/number.
- [ ] Client/dependency dùng pointer.
- [ ] Method I/O nhận `context.Context`.
- [ ] Thêm errors.go / util.go chỉ khi thật cần.

---

## 7. Tham chiếu nhanh

| Pkg    | Interface | Impl       | Config                        | Ghi chú                |
| ------ | --------- | ---------- | ----------------------------- | ---------------------- |
| gemini | IGemini   | geminiImpl | GeminiConfig                  | 4 file                 |
| voyage | IVoyage   | voyageImpl | VoyageConfig                  | 4 file                 |
| redis  | IRedis    | redisImpl  | RedisConfig (+ Config alias)  | 4 file + errors        |
| qdrant | IQdrant   | qdrantImpl | QdrantConfig (+ Config alias) | 4 file + errors + util |

---

_Cập nhật theo các lần refactor gemini, voyage, redis, qdrant._
