# ER diagram

```mermaid
erDiagram
  HOSPITALS ||--o{ STAFF : employs
  HOSPITALS ||--o{ PATIENTS : owns
  HOSPITALS {
    uuid id PK
    varchar code UK
    varchar name
  }
  STAFF {
    uuid id PK
    varchar username
    text password_hash
    uuid hospital_id FK
  }
  PATIENTS {
    uuid id PK
    uuid hospital_id FK
    varchar national_id
    varchar passport_id
    varchar first_name_th
    varchar middle_name_th
    varchar last_name_th
    varchar first_name_en
    varchar middle_name_en
    varchar last_name_en
    date date_of_birth
    varchar phone_number
    varchar email
    char gender
  }
```

`patients` requires either `national_id` or `passport_id`. Both identifiers are unique only inside a hospital, supporting separate HIS domains while staff authorization always limits access to the associated hospital.
