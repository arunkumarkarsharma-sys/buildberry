Generic DB Import/Export System in Go

Overview:-

This project provides dynamic CSV Export,CSV Import,and Excel import api's using Go
The system dynamically reads database schema,Validate column types,handle transactions,and support generic table-based operations.

Features:-

1) Dynamic schema reader.
2) CSV Export
3) CSV Import
4) Excel Import
5) Type conversion
6) Required filled validation
7) Unknown column validation
8) Transaction support
9) Row-level error tracking
10) Generic reusable APIs


Techstack:-

1) Golang
2) Mysql
3) postman


Api Endpoint:-


Health check:
http://localhost:5000/health

Test schema api:
GET http://localhost:5000/api/schema/users

CSV Export:
GET http://localhost:5000/api/export/users.csv

CSV Import:
POST http://localhost:5000/api/import/users/csv
Body → form-data

file = users.csv

Excel Import:
POST http://localhost:5000/api/import-excel/users
Body → form-data

file = users.xlsx


Testing:-

1) Tested CSV import
2) Tested CSV export
3) Tested Excel import
4) Tested invalid table validation
5) Tested missing required column validation
6) Tested wrong datatype validation
7) Tested duplicate key handling
