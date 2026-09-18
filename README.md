# Agnos Test

ระบบ Back-end สำหรับเป็นตัวกลางระหว่างผู้ใช้งานของโรงพยาบาลกับข้อมูลผู้ป่วย (Hospital Information System: HIS) โดยออกแบบด้วย Go, Gin และ PostgreSQL

## ระบบนี้มีอะไรบ้าง

- API สร้างบัญชีเจ้าหน้าที่ `POST /staff/create` โดย username จะไม่ซ้ำกันภายในโรงพยาบาลเดียวกัน
- API เข้าสู่ระบบ `POST /staff/login` และออก JWT อายุ 8 ชั่วโมง
- API ค้นหาผู้ป่วย `GET /patient/search` รองรับ national ID, passport ID, ชื่อ, วันเกิด, เบอร์โทร และอีเมล
- การจำกัดสิทธิ์ตามโรงพยาบาล: ระบบอ่าน `hospital_id` จาก JWT เท่านั้น จึงค้นหาข้อมูลข้ามโรงพยาบาลไม่ได้
- โมเดลข้อมูล Hospital, Staff และ Patient ที่รองรับชื่อไทย/อังกฤษ, national ID หรือ passport ID, วันเกิด, เพศ และข้อมูลติดต่อ
- PostgreSQL schema พร้อม foreign keys, unique indexes และ validation ของเพศ/ตัวระบุผู้ป่วย
- Nginx reverse proxy พร้อม rate limit, Docker Compose สำหรับ Nginx + Go API + PostgreSQL
- Unit tests สำหรับกรณีสำเร็จและล้มเหลวของ API หลัก รวมถึงตรวจสอบ authorization scope

## ความปลอดภัย

- รหัสผ่านถูก hash ด้วย bcrypt และไม่ถูกส่งกลับจาก API
- JWT ใช้ HMAC และตรวจสอบอายุ token ก่อนอนุญาตให้ค้นหาผู้ป่วย
- JWT ระบุ issuer และ audience, ใช้เฉพาะ HS256 และรองรับ logout เพื่อ revoke token ก่อนหมดอายุ
- การสร้าง staff ต้องส่ง `X-Provisioning-Key` ที่ตรงกับ `PROVISIONING_API_KEY`; endpoint นี้จึงไม่เปิดให้บุคคลทั่วไปสร้างบัญชี
- access log บันทึกเฉพาะ method, route, status และระยะเวลา โดยไม่บันทึก query string ที่อาจมี PII
- จำกัด request body ที่ 1 MiB และตั้ง read/write/idle timeouts ที่ HTTP server
- บันทึก audit event ทุกครั้งที่ค้นหาผู้ป่วย โดยเก็บ staff, hospital, ชนิดของเงื่อนไขที่ใช้ และจำนวนผลลัพธ์ แต่ไม่เก็บค่าข้อมูลผู้ป่วยดิบ
- จำกัดจำนวนรายการผลลัพธ์ที่ 100 รายการต่อคำขอ
- ควรเก็บ `JWT_SECRET` ใน secret manager, เปิด TLS ที่ ingress และหลีกเลี่ยงการ log ข้อมูลผู้ป่วยจริงใน production

## เริ่มใช้งาน

```powershell
Copy-Item .env.example .env
docker compose up --build
```

API จะให้บริการที่ `http://localhost:8080` โปรดเปลี่ยน `JWT_SECRET` ใน `.env` ก่อนใช้งานนอกเครื่องพัฒนา

## การตั้งค่า `GIN_MODE`

`GIN_MODE` เป็น environment variable ที่กำหนดโหมดทำงานของ Gin framework

- `debug` - เหมาะกับการพัฒนา แสดง log และรายละเอียด route มากขึ้น
- `release` - เหมาะกับ production ลด log และไม่แสดงข้อมูล debug ที่ไม่จำเป็น
- `test` - ใช้สำหรับ automated tests

ไฟล์ `.env.example` ตั้งค่าเป็น `GIN_MODE=release` เพื่อให้เหมาะกับการรันผ่าน Docker และ Nginx ในลักษณะใกล้ production หากกำลังพัฒนาและต้องการดูรายละเอียดเพิ่ม ให้เปลี่ยนเป็น `GIN_MODE=debug`

## Environment variables เพิ่มเติม

- `JWT_ISSUER` และ `JWT_AUDIENCE` - ค่าที่ต้องตรงกันเมื่อออกและตรวจสอบ JWT
- `PROVISIONING_API_KEY` - secret ความยาวอย่างน้อย 32 ตัวอักษร สำหรับระบบภายในที่เรียก `POST /staff/create`; ส่งผ่าน header `X-Provisioning-Key`

ห้าม commit ค่า secret จริง และควรเปลี่ยน `JWT_SECRET` เพื่อหมุน signing key ตามนโยบายของระบบ หากหมุน key แล้ว token เก่าจะใช้ไม่ได้โดยตั้งใจ

## ทดสอบ

```powershell
go test ./...
```

ดูรายละเอียดเพิ่มได้ที่ [แผนและโครงสร้างระบบ](docs/planning.md), [API specification](docs/api.md) และ [ER diagram](docs/er-diagram.md)
