# Fuzzy report

| | |
|---|---|
| **Target** | http://localhost:8000 |
| **Spec** | ./examples/crud/artifacts/openapi.json (Fuzzy CRUD example 1.0.0) |
| **Dictionary** | none (random values) |
| **Random bounds** | string length 1..10 (unicode), int 0..1000, float 0..1000 |
| **Status filter** | all |
| **Seed** | 6055143362844996739 |
| **Started** | 2026-11-03T01:45:32+01:00 |
| **Duration** | 25ms |
| **Requests sent** | 500 |
| **Results reported** | 500 |

## Summary

| Endpoint | Sent | Reported | Status codes |
|---|---|---|---|
| GET /items | 100 | 100 | 200 ×100 |
| POST /items | 100 | 100 | 201 ×100 |
| GET /items/{item\_id} | 100 | 100 | 200 ×11, 404 ×89 |
| PUT /items/{item\_id} | 100 | 100 | 200 ×14, 404 ×86 |
| DELETE /items/{item\_id} | 100 | 100 | 204 ×16, 404 ×84 |

## GET /items

List Items

- **HTTP verb:** GET
- **Path:** /items
- **Operation ID:** list\_items\_items\_get

### Parameters

| Name | In | Type | Required | Constraints |
|---|---|---|---|---|
| limit | query | integer | no | min 1; max 100 |
| offset | query | integer | no | min 0 |

### Results

100 sent, 100 reported (filter: all). Status codes: 200 ×100.

| Endpoint | Status | Message | Input |
|---|---|---|---|
| GET /items | 200 | \[\] | limit=61, offset=157 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[\] | offset=757 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=80 |
| GET /items | 200 | \[\] | limit=81, offset=282 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[\] | limit=45, offset=919 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[\] | offset=507 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[\] | offset=810 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[\] | offset=747 |
| GET /items | 200 | \[\] | limit=76, offset=807 |
| GET /items | 200 | \[{"name":"𠀦z값ば","description":null,"price":293.209,"quantity":510,"id":53},{"name":"𠀧Ñ","description":null,"price":196.969,"quantity":977,"id":54},{"name":"𠀙","description":null,"price":914.573,"quant… | offset=48 |
| GET /items | 200 | \[\] | offset=802 |
| GET /items | 200 | \[{"name":"갸к𠁎𠀳るぢ丱","description":"θ감","price":979.609,"quantity":835,"id":40},{"name":"ΑЖ가","description":"Âऐ걏びشक","price":19.686,"quantity":358,"id":42},{"name":"갔んγå갭","description":null,"price":562… | offset=39 |
| GET /items | 200 | \[\] | offset=223 |
| GET /items | 200 | \[{"name":"业갿उ","description":null,"price":481.185,"quantity":643,"id":80},{"name":"𠀲ऌЕ","description":"世並ê갛こ갺𠁟书H객","price":848.374,"quantity":0,"id":81},{"name":"ΰ","description":null,"price":990.331,… | offset=73 |
| GET /items | 200 | \[\] | limit=99, offset=949 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=69 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=93 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=98 |
| GET /items | 200 | \[\] | limit=62, offset=331 |
| GET /items | 200 | \[\] | limit=7, offset=268 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=25 |
| GET /items | 200 | \[\] | limit=94, offset=815 |
| GET /items | 200 | \[\] | offset=621 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[\] | limit=96, offset=832 |
| GET /items | 200 | \[\] | offset=581 |
| GET /items | 200 | \[\] | offset=323 |
| GET /items | 200 | \[\] | offset=329 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=93 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=73 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=47 |
| GET /items | 200 | \[\] | offset=596 |
| GET /items | 200 | \[\] | offset=136 |
| GET /items | 200 | \[\] | limit=62, offset=270 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[\] | limit=46, offset=472 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=82 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[\] | limit=97, offset=797 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[\] | limit=57, offset=242 |
| GET /items | 200 | \[\] | offset=657 |
| GET /items | 200 | \[\] | offset=307 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[\] | limit=75, offset=775 |
| GET /items | 200 | \[\] | limit=90, offset=742 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[\] | offset=211 |
| GET /items | 200 | \[\] | offset=579 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=52 |
| GET /items | 200 | \[\] | offset=726 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=65 |
| GET /items | 200 | \[\] | limit=41, offset=886 |
| GET /items | 200 | \[\] | offset=458 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=95 |
| GET /items | 200 | \[\] | offset=511 |
| GET /items | 200 | \[\] | offset=554 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=27 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=7 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=10 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=28 |
| GET /items | 200 | \[\] | offset=405 |
| GET /items | 200 | \[\] | offset=258 |
| GET /items | 200 | \[\] | limit=91, offset=996 |
| GET /items | 200 | \[\] | limit=82, offset=979 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[{"name":"そ걵","description":"习शك丧갲ळر","price":794.961,"quantity":988,"id":6},{"name":"झ乯갽𠀰业","description":null,"price":73.135,"quantity":540,"id":8},{"name":"갽𠀩丌主𠀽","description":null,"price":511.217… | limit=83, offset=4 |
| GET /items | 200 | \[{"name":"걲𠀫乹严乙𠁹ΔهΘ갯","description":null,"price":914.378,"quantity":0,"id":27},{"name":"ग主АÈة𠁮B","description":"𠁵","price":859.922,"quantity":0,"id":28},{"name":"걹Тo걬H买","description":null,"price":373… | offset=22 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=97 |
| GET /items | 200 | \[\] | limit=85, offset=998 |
| GET /items | 200 | \[\] | offset=224 |
| GET /items | 200 | \[\] | offset=220 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=5 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=28 |
| GET /items | 200 | \[\] | offset=602 |
| GET /items | 200 | \[\] | offset=218 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=61 |
| GET /items | 200 | \[{"name":"Ï𠁓丢","description":null,"price":404.225,"quantity":0,"id":122},{"name":"ßÝ乆Β갗𠀪غ𠀳","description":null,"price":387.191,"quantity":0,"id":124},{"name":"𠁀","description":null,"price":827.648,"qu… | offset=104 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=78 |
| GET /items | 200 | \[\] | offset=493 |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… |  |
| GET /items | 200 | \[{"name":"丶ΩçЭ","description":"乖값主δΦÍ","price":567.326,"quantity":248,"id":1},{"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2},{"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.9… | limit=71 |
| GET /items | 200 | \[\] | offset=742 |
| GET /items | 200 | \[\] | offset=353 |
| GET /items | 200 | \[\] | limit=68, offset=376 |

## POST /items

Create Item

- **HTTP verb:** POST
- **Path:** /items
- **Operation ID:** create\_item\_items\_post

### Parameters

| Name | In | Type | Required | Constraints |
|---|---|---|---|---|
| (body) | application/json | object | yes |  |
| description | application/json | string (nullable) | no | max length 256 |
| name | application/json | string | yes | min length 1; max length 64 |
| price | application/json | number | yes | exclusive min 0; max 10000 |
| quantity | application/json | integer | no | min 0; max 1000 |

### Results

100 sent, 100 reported (filter: all). Status codes: 201 ×100.

| Endpoint | Status | Message | Input |
|---|---|---|---|
| POST /items | 201 | {"name":"Υ갭𠀂갺み","description":"𠁉갺ई𠀣걋3𠁕","price":325.438,"quantity":0,"id":102} | body={"description":"𠁉갺ई𠀣걋3𠁕","name":"Υ갭𠀂갺み","price":325.438} |
| POST /items | 201 | {"name":"𠀠ぅáЯΛ갳","description":null,"price":993.511,"quantity":0,"id":101} | body={"name":"𠀠ぅáЯΛ갳","price":993.511} |
| POST /items | 201 | {"name":"ß","description":null,"price":649.099,"quantity":303,"id":104} | body={"name":"ß","price":649.099,"quantity":303} |
| POST /items | 201 | {"name":"𠀫ょÏ","description":null,"price":798.656,"quantity":0,"id":103} | body={"name":"𠀫ょÏ","price":798.656} |
| POST /items | 201 | {"name":"갴𠁱V","description":null,"price":669.533,"quantity":859,"id":105} | body={"name":"갴𠁱V","price":669.533,"quantity":859} |
| POST /items | 201 | {"name":"ぇす","description":null,"price":984.839,"quantity":0,"id":106} | body={"name":"ぇす","price":984.839} |
| POST /items | 201 | {"name":"𠀜𠁤Óゖ丂걽pどeç","description":null,"price":666.053,"quantity":922,"id":107} | body={"name":"𠀜𠁤Óゖ丂걽pどeç","price":666.053,"quantity":922} |
| POST /items | 201 | {"name":"갾","description":null,"price":567.774,"quantity":663,"id":108} | body={"name":"갾","price":567.774,"quantity":663} |
| POST /items | 201 | {"name":"ぇχ걀χΔ𠁍л乼𠀀","description":null,"price":107.802,"quantity":0,"id":110} | body={"name":"ぇχ걀χΔ𠁍л乼𠀀","price":107.802} |
| POST /items | 201 | {"name":"갫𠁃i갚E걫","description":null,"price":85.721,"quantity":741,"id":109} | body={"name":"갫𠁃i갚E걫","price":85.721,"quantity":741} |
| POST /items | 201 | {"name":"걼거зп乚且","description":"यめ가𠀋乎बضЪउ","price":986.858,"quantity":0,"id":112} | body={"description":"यめ가𠀋乎बضЪउ","name":"걼거зп乚且","price":986.858} |
| POST /items | 201 | {"name":"𠁾감छたèT","description":"갔ど𠀜اт","price":50.685,"quantity":0,"id":111} | body={"description":"갔ど𠀜اт","name":"𠁾감छたèT","price":50.685} |
| POST /items | 201 | {"name":"ؿ丁久𠁏W갊Æظ","description":"乇걅买Ôъz개","price":436.163,"quantity":552,"id":113} | body={"description":"乇걅买Ôъz개","name":"ؿ丁久𠁏W갊Æظ","price":436.163,"quantity":552} |
| POST /items | 201 | {"name":"ρ걸é𠁺","description":"两Йц𠁪ô걑èう","price":868.289,"quantity":186,"id":115} | body={"description":"两Йц𠁪ô걑èう","name":"ρ걸é𠁺","price":868.289,"quantity":186} |
| POST /items | 201 | {"name":"𠀿ÿЕ걢ぞμ","description":"丿ょ","price":789.179,"quantity":941,"id":114} | body={"description":"丿ょ","name":"𠀿ÿЕ걢ぞμ","price":789.179,"quantity":941} |
| POST /items | 201 | {"name":"𠀉丄𠀺り𠀪l乥","description":null,"price":503.676,"quantity":0,"id":116} | body={"name":"𠀉丄𠀺り𠀪l乥","price":503.676} |
| POST /items | 201 | {"name":"ずЛ𠁒义𠁃छ𠀬𠁛","description":"𠀩","price":629.657,"quantity":942,"id":118} | body={"description":"𠀩","name":"ずЛ𠁒义𠁃छ𠀬𠁛","price":629.657,"quantity":942} |
| POST /items | 201 | {"name":"갻kبΨ丿","description":null,"price":832.325,"quantity":0,"id":117} | body={"name":"갻kبΨ丿","price":832.325} |
| POST /items | 201 | {"name":"Å𠀂х","description":null,"price":874.836,"quantity":204,"id":120} | body={"name":"Å𠀂х","price":874.836,"quantity":204} |
| POST /items | 201 | {"name":"ぅ𠁉𠀚","description":null,"price":146.848,"quantity":697,"id":119} | body={"name":"ぅ𠁉𠀚","price":146.848,"quantity":697} |
| POST /items | 201 | {"name":"걙C𠀁Εχß","description":"七上𠁴걕갅𠀶ぶ걯갥ぺ","price":404.87,"quantity":672,"id":121} | body={"description":"七上𠁴걕갅𠀶ぶ걯갥ぺ","name":"걙C𠀁Εχß","price":404.87,"quantity":672} |
| POST /items | 201 | {"name":"Ï𠁓丢","description":null,"price":404.225,"quantity":0,"id":122} | body={"name":"Ï𠁓丢","price":404.225} |
| POST /items | 201 | {"name":"걽걀حあ걌र个べ","description":"ؾطбΛëـ𠁒ôΠ","price":163.054,"quantity":911,"id":123} | body={"description":"ؾطбΛëـ𠁒ôΠ","name":"걽걀حあ걌र个べ","price":163.054,"quantity":911} |
| POST /items | 201 | {"name":"ßÝ乆Β갗𠀪غ𠀳","description":null,"price":387.191,"quantity":0,"id":124} | body={"name":"ßÝ乆Β갗𠀪غ𠀳","price":387.191} |
| POST /items | 201 | {"name":"𠁀","description":null,"price":827.648,"quantity":612,"id":125} | body={"name":"𠁀","price":827.648,"quantity":612} |
| POST /items | 201 | {"name":"걚び𠁘","description":"𠁅𠀽갊乪𠁻","price":935.321,"quantity":0,"id":126} | body={"description":"𠁅𠀽갊乪𠁻","name":"걚び𠁘","price":935.321} |
| POST /items | 201 | {"name":"乇5串Оめ갸оだ𠀳e","description":"𠀬乱らح乑ゖà𠁃","price":330.903,"quantity":642,"id":127} | body={"description":"𠀬乱らح乑ゖà𠁃","name":"乇5串Оめ갸оだ𠀳e","price":330.903,"quantity":642} |
| POST /items | 201 | {"name":"𠁚𠀙갪ぞТ丽8Ϋ𠁅","description":"걃UΦؿ","price":947.882,"quantity":0,"id":131} | body={"description":"걃UΦؿ","name":"𠁚𠀙갪ぞТ丽8Ϋ𠁅","price":947.882} |
| POST /items | 201 | {"name":"𠁗걼жЗضむжмí","description":null,"price":15.658,"quantity":430,"id":128} | body={"name":"𠁗걼жЗضむжмí","price":15.658,"quantity":430} |
| POST /items | 201 | {"name":"𠁣èز같","description":null,"price":456.123,"quantity":56,"id":130} | body={"name":"𠁣èز같","price":456.123,"quantity":56} |
| POST /items | 201 | {"name":"ぐЪ业𠁜𠀟𠀱σ丠걶و","description":null,"price":491.938,"quantity":875,"id":129} | body={"description":null,"name":"ぐЪ业𠁜𠀟𠀱σ丠걶و","price":491.938,"quantity":875} |
| POST /items | 201 | {"name":"έغ乢丅갓","description":"ζ3걷","price":739.657,"quantity":0,"id":132} | body={"description":"ζ3걷","name":"έغ乢丅갓","price":739.657} |
| POST /items | 201 | {"name":"久ع갍𠁼乭걍","description":null,"price":662.604,"quantity":0,"id":134} | body={"name":"久ع갍𠁼乭걍","price":662.604} |
| POST /items | 201 | {"name":"걁갹ぴÞद丄","description":"갰õで𠁨","price":772.476,"quantity":0,"id":133} | body={"description":"갰õで𠁨","name":"걁갹ぴÞद丄","price":772.476} |
| POST /items | 201 | {"name":"äû걷षMΦ","description":"ν걠갗𠁘حý专ぶO","price":429.083,"quantity":93,"id":135} | body={"description":"ν걠갗𠁘حý专ぶO","name":"äû걷षMΦ","price":429.083,"quantity":93} |
| POST /items | 201 | {"name":"丶𠁢","description":null,"price":599.065,"quantity":841,"id":136} | body={"name":"丶𠁢","price":599.065,"quantity":841} |
| POST /items | 201 | {"name":"걧","description":null,"price":899.553,"quantity":145,"id":137} | body={"name":"걧","price":899.553,"quantity":145} |
| POST /items | 201 | {"name":"ф갪𠀈걵Äゐあ걬ऴ","description":"丽걗Ыà걼书r갑z𠀺","price":520.125,"quantity":963,"id":138} | body={"description":"丽걗Ыà걼书r갑z𠀺","name":"ф갪𠀈걵Äゐあ걬ऴ","price":520.125,"quantity":963} |
| POST /items | 201 | {"name":"اâउΟず","description":"ず𠀹ؿΠÃP거θ","price":527.846,"quantity":0,"id":139} | body={"description":"ず𠀹ؿΠÃP거θ","name":"اâउΟず","price":527.846} |
| POST /items | 201 | {"name":"ちぬë乎","description":null,"price":179.782,"quantity":0,"id":140} | body={"name":"ちぬë乎","price":179.782} |
| POST /items | 201 | {"name":"ý𠁎MW","description":"𠁢ड乇Îई","price":32.867,"quantity":83,"id":141} | body={"description":"𠁢ड乇Îई","name":"ý𠁎MW","price":32.867,"quantity":83} |
| POST /items | 201 | {"name":"𠀋举ञधцß걊u걲","description":"つऩáþmΧञ𠀳","price":167.586,"quantity":0,"id":142} | body={"description":"つऩáþmΧञ𠀳","name":"𠀋举ञधцß걊u걲","price":167.586} |
| POST /items | 201 | {"name":"ड갔ى丷걑𠁕Æ","description":null,"price":206.205,"quantity":0,"id":147} | body={"name":"ड갔ى丷걑𠁕Æ","price":206.205} |
| POST /items | 201 | {"name":"ÿΓ𠁢","description":null,"price":295.889,"quantity":129,"id":143} | body={"name":"ÿΓ𠁢","price":295.889,"quantity":129} |
| POST /items | 201 | {"name":"강걮ΰنÍø","description":null,"price":656.349,"quantity":0,"id":144} | body={"name":"강걮ΰنÍø","price":656.349} |
| POST /items | 201 | {"name":"ほ𠁸갳ΫΧ丟つ","description":"丐","price":151.129,"quantity":0,"id":145} | body={"description":"丐","name":"ほ𠁸갳ΫΧ丟つ","price":151.129} |
| POST /items | 201 | {"name":"𠀵ó","description":"Л걡갪걎𠀡Á갠乄","price":313.4,"quantity":0,"id":146} | body={"description":"Л걡갪걎𠀡Á갠乄","name":"𠀵ó","price":313.4} |
| POST /items | 201 | {"name":"Ιd","description":"ずб下W갢乮","price":982.362,"quantity":365,"id":148} | body={"description":"ずб下W갢乮","name":"Ιd","price":982.362,"quantity":365} |
| POST /items | 201 | {"name":"𠀜ضπr乯串乏","description":"Õ丼おθέれì","price":142.627,"quantity":199,"id":149} | body={"description":"Õ丼おθέれì","name":"𠀜ضπr乯串乏","price":142.627,"quantity":199} |
| POST /items | 201 | {"name":"ई乤之𠁤걸ゃú乽ऑ","description":"ら","price":3.322,"quantity":949,"id":150} | body={"description":"ら","name":"ई乤之𠁤걸ゃú乽ऑ","price":3.322,"quantity":949} |
| POST /items | 201 | {"name":"भrら丙걾रÇ上","description":"Ú갿Γ乾걜Θ","price":512.607,"quantity":320,"id":151} | body={"description":"Ú갿Γ乾걜Θ","name":"भrら丙걾रÇ上","price":512.607,"quantity":320} |
| POST /items | 201 | {"name":"乹こ𠁫𠁱","description":"걙갑걪갳걙","price":795.511,"quantity":646,"id":152} | body={"description":"걙갑걪갳걙","name":"乹こ𠁫𠁱","price":795.511,"quantity":646} |
| POST /items | 201 | {"name":"ふ갛óوきн𠀙","description":null,"price":783.963,"quantity":0,"id":153} | body={"name":"ふ갛óوきн𠀙","price":783.963} |
| POST /items | 201 | {"name":"𠁎दこ걳ي乜Цग丝","description":null,"price":796.448,"quantity":285,"id":154} | body={"name":"𠁎दこ걳ي乜Цग丝","price":796.448,"quantity":285} |
| POST /items | 201 | {"name":"ぺû걾よП","description":"г갣","price":187.993,"quantity":0,"id":155} | body={"description":"г갣","name":"ぺû걾よП","price":187.993} |
| POST /items | 201 | {"name":"乙같の丼Ç","description":"갳Ò个ν걵𠀄","price":186.119,"quantity":750,"id":156} | body={"description":"갳Ò个ν걵𠀄","name":"乙같の丼Ç","price":186.119,"quantity":750} |
| POST /items | 201 | {"name":"め万乻へइ乪ï𠀙갪𠀹","description":null,"price":767.155,"quantity":0,"id":158} | body={"name":"め万乻へइ乪ï𠀙갪𠀹","price":767.155} |
| POST /items | 201 | {"name":"𠁨가W义ण𠁉झ","description":"Ìび𠀼东","price":945.672,"quantity":667,"id":157} | body={"description":"Ìび𠀼东","name":"𠁨가W义ण𠁉झ","price":945.672,"quantity":667} |
| POST /items | 201 | {"name":"乢걃à𠀬थ𠀔","description":"갍","price":580.44,"quantity":0,"id":160} | body={"description":"갍","name":"乢걃à𠀬थ𠀔","price":580.44} |
| POST /items | 201 | {"name":"갬𠁎Z걑乃","description":null,"price":6.466,"quantity":0,"id":159} | body={"name":"갬𠁎Z걑乃","price":6.466} |
| POST /items | 201 | {"name":"त𠀪","description":"걻れ갯स","price":450.617,"quantity":388,"id":162} | body={"description":"걻れ갯स","name":"त𠀪","price":450.617,"quantity":388} |
| POST /items | 201 | {"name":"丒sFt","description":"乩ぬ갈Ф걂","price":72.962,"quantity":0,"id":161} | body={"description":"乩ぬ갈Ф걂","name":"丒sFt","price":72.962} |
| POST /items | 201 | {"name":"𠁯𠁵𠀭ом","description":null,"price":167.253,"quantity":8,"id":163} | body={"name":"𠁯𠁵𠀭ом","price":167.253,"quantity":8} |
| POST /items | 201 | {"name":"ま걕","description":null,"price":580.06,"quantity":740,"id":164} | body={"name":"ま걕","price":580.06,"quantity":740} |
| POST /items | 201 | {"name":"乂औ값स𠁐","description":"δぬ𠀻갎걖","price":817.337,"quantity":259,"id":166} | body={"description":"δぬ𠀻갎걖","name":"乂औ값स𠁐","price":817.337,"quantity":259} |
| POST /items | 201 | {"name":"Λえ丈टЖ𠀳걄Ò","description":null,"price":183.622,"quantity":290,"id":165} | body={"name":"Λえ丈टЖ𠀳걄Ò","price":183.622,"quantity":290} |
| POST /items | 201 | {"name":"𠀤ؽй","description":null,"price":635.913,"quantity":0,"id":167} | body={"name":"𠀤ؽй","price":635.913} |
| POST /items | 201 | {"name":"u5","description":"òые並ば","price":471.418,"quantity":423,"id":168} | body={"description":"òые並ば","name":"u5","price":471.418,"quantity":423} |
| POST /items | 201 | {"name":"갚たお𠀞f乧6Ñ","description":"φ걖л갾ऐ九ऴ乛丁ð","price":877.78,"quantity":0,"id":169} | body={"description":"φ걖л갾ऐ九ऴ乛丁ð","name":"갚たお𠀞f乧6Ñ","price":877.78} |
| POST /items | 201 | {"name":"И乾걩갚乹걙","description":null,"price":653.962,"quantity":129,"id":170} | body={"name":"И乾걩갚乹걙","price":653.962,"quantity":129} |
| POST /items | 201 | {"name":"оν严乴त","description":null,"price":277.423,"quantity":356,"id":171} | body={"description":null,"name":"оν严乴त","price":277.423,"quantity":356} |
| POST /items | 201 | {"name":"丽丑걽","description":"之þم7乛â","price":709.925,"quantity":0,"id":172} | body={"description":"之þم7乛â","name":"丽丑걽","price":709.925} |
| POST /items | 201 | {"name":"ゐ","description":null,"price":769.542,"quantity":392,"id":173} | body={"name":"ゐ","price":769.542,"quantity":392} |
| POST /items | 201 | {"name":"Ι걭𠀱𠀯ょб𠀽","description":null,"price":737.205,"quantity":262,"id":174} | body={"name":"Ι걭𠀱𠀯ょб𠀽","price":737.205,"quantity":262} |
| POST /items | 201 | {"name":"öष갇𠀗z","description":null,"price":728.53,"quantity":0,"id":176} | body={"name":"öष갇𠀗z","price":728.53} |
| POST /items | 201 | {"name":"चभ","description":"я乵𠀠걫Àk갭걋专","price":829.154,"quantity":0,"id":175} | body={"description":"я乵𠀠걫Àk갭걋专","name":"चभ","price":829.154} |
| POST /items | 201 | {"name":"丿갾걨","description":"qبرå乀か걖ऐ","price":252.359,"quantity":311,"id":177} | body={"description":"qبرå乀か걖ऐ","name":"丿갾걨","price":252.359,"quantity":311} |
| POST /items | 201 | {"name":"ا业걫ょO갧","description":"两ÿ걂きM","price":519.052,"quantity":0,"id":178} | body={"description":"两ÿ걂きM","name":"ا业걫ょO갧","price":519.052} |
| POST /items | 201 | {"name":"ñ𠁳φ","description":"3七","price":912.35,"quantity":0,"id":179} | body={"description":"3七","name":"ñ𠁳φ","price":912.35} |
| POST /items | 201 | {"name":"Â걀छU","description":null,"price":858.265,"quantity":0,"id":180} | body={"name":"Â걀छU","price":858.265} |
| POST /items | 201 | {"name":"D","description":null,"price":827.866,"quantity":0,"id":181} | body={"name":"D","price":827.866} |
| POST /items | 201 | {"name":"𠁾걧ذcق","description":null,"price":493.431,"quantity":0,"id":183} | body={"description":null,"name":"𠁾걧ذcق","price":493.431} |
| POST /items | 201 | {"name":"फض乓ج丞फ专𠀟乻","description":null,"price":934.063,"quantity":166,"id":182} | body={"name":"फض乓ج丞फ专𠀟乻","price":934.063,"quantity":166} |
| POST /items | 201 | {"name":"之t𠁎𠁕不へΦゃ","description":null,"price":997.131,"quantity":145,"id":185} | body={"name":"之t𠁎𠁕不へΦゃ","price":997.131,"quantity":145} |
| POST /items | 201 | {"name":"Êضぱ𠀓","description":null,"price":649.808,"quantity":432,"id":184} | body={"name":"Êضぱ𠀓","price":649.808,"quantity":432} |
| POST /items | 201 | {"name":"𠀚ς𠁫y걈","description":null,"price":700.973,"quantity":111,"id":186} | body={"name":"𠀚ς𠁫y걈","price":700.973,"quantity":111} |
| POST /items | 201 | {"name":"걥ぞΟऱ𠀰","description":"χ𠀡ञ","price":763.389,"quantity":0,"id":187} | body={"description":"χ𠀡ञ","name":"걥ぞΟऱ𠀰","price":763.389} |
| POST /items | 201 | {"name":"갓","description":null,"price":567.761,"quantity":0,"id":189} | body={"name":"갓","price":567.761} |
| POST /items | 201 | {"name":"𠁉","description":null,"price":594.31,"quantity":0,"id":188} | body={"name":"𠁉","price":594.31} |
| POST /items | 201 | {"name":"õ걃ぁ乙ΝOも","description":"丽","price":917.248,"quantity":402,"id":190} | body={"description":"丽","name":"õ걃ぁ乙ΝOも","price":917.248,"quantity":402} |
| POST /items | 201 | {"name":"𠁮之ऐΘडへΤ","description":"öъき","price":622.305,"quantity":554,"id":191} | body={"description":"öъき","name":"𠁮之ऐΘडへΤ","price":622.305,"quantity":554} |
| POST /items | 201 | {"name":"걫갷8乓","description":null,"price":574.574,"quantity":0,"id":192} | body={"description":null,"name":"걫갷8乓","price":574.574} |
| POST /items | 201 | {"name":"à両","description":null,"price":864.752,"quantity":0,"id":193} | body={"name":"à両","price":864.752} |
| POST /items | 201 | {"name":"𠀦बΨ","description":"𠀠之𠁵حqΝ걟ΡÌ","price":940.301,"quantity":0,"id":194} | body={"description":"𠀠之𠁵حqΝ걟ΡÌ","name":"𠀦बΨ","price":940.301} |
| POST /items | 201 | {"name":"𠁹Ы갴ゃ𠁨ؿ","description":null,"price":467.515,"quantity":916,"id":195} | body={"name":"𠁹Ы갴ゃ𠁨ؿ","price":467.515,"quantity":916} |
| POST /items | 201 | {"name":"𠁖𠀛Ω","description":null,"price":761.767,"quantity":720,"id":196} | body={"name":"𠁖𠀛Ω","price":761.767,"quantity":720} |
| POST /items | 201 | {"name":"万걅","description":null,"price":630.857,"quantity":0,"id":197} | body={"name":"万걅","price":630.857} |
| POST /items | 201 | {"name":"इ𠀻á𠁥丙Íぷश","description":null,"price":656.068,"quantity":0,"id":198} | body={"name":"इ𠀻á𠁥丙Íぷश","price":656.068} |
| POST /items | 201 | {"name":"ä걇ИaÔ갿乸乂じ","description":"𠁝九𠁵ば𠁁Ο갋乓下A","price":340.617,"quantity":0,"id":199} | body={"description":"𠁝九𠁵ば𠁁Ο갋乓下A","name":"ä걇ИaÔ갿乸乂じ","price":340.617} |
| POST /items | 201 | {"name":"ه갿","description":null,"price":564.538,"quantity":0,"id":200} | body={"name":"ه갿","price":564.538} |

## GET /items/{item\_id}

Get Item

- **HTTP verb:** GET
- **Path:** /items/{item\_id}
- **Operation ID:** get\_item\_items\_\_item\_id\_\_get

### Parameters

| Name | In | Type | Required | Constraints |
|---|---|---|---|---|
| item\_id | path | integer | yes | exclusive min 0 |

### Results

100 sent, 100 reported (filter: all). Status codes: 200 ×11, 404 ×89.

| Endpoint | Status | Message | Input |
|---|---|---|---|
| GET /items/482 | 404 | {"detail":"item 482 not found"} | item\_id=482 |
| GET /items/32 | 200 | {"name":"り걢","description":null,"price":274.045,"quantity":533,"id":32} | item\_id=32 |
| GET /items/72 | 200 | {"name":"𠀌𠀾ぶ갲乳うщ𠀄طا","description":"걛乳丞","price":350.919,"quantity":0,"id":72} | item\_id=72 |
| GET /items/183 | 404 | {"detail":"item 183 not found"} | item\_id=183 |
| GET /items/3 | 200 | {"name":"ΰ主𠀴ゃæ乪Α𠁭","description":null,"price":448.99,"quantity":674,"id":3} | item\_id=3 |
| GET /items/888 | 404 | {"detail":"item 888 not found"} | item\_id=888 |
| GET /items/180 | 404 | {"detail":"item 180 not found"} | item\_id=180 |
| GET /items/942 | 404 | {"detail":"item 942 not found"} | item\_id=942 |
| GET /items/252 | 404 | {"detail":"item 252 not found"} | item\_id=252 |
| GET /items/264 | 404 | {"detail":"item 264 not found"} | item\_id=264 |
| GET /items/80 | 200 | {"name":"业갿उ","description":null,"price":481.185,"quantity":643,"id":80} | item\_id=80 |
| GET /items/400 | 404 | {"detail":"item 400 not found"} | item\_id=400 |
| GET /items/118 | 404 | {"detail":"item 118 not found"} | item\_id=118 |
| GET /items/518 | 404 | {"detail":"item 518 not found"} | item\_id=518 |
| GET /items/611 | 404 | {"detail":"item 611 not found"} | item\_id=611 |
| GET /items/212 | 404 | {"detail":"item 212 not found"} | item\_id=212 |
| GET /items/991 | 404 | {"detail":"item 991 not found"} | item\_id=991 |
| GET /items/28 | 200 | {"name":"ग主АÈة𠁮B","description":"𠁵","price":859.922,"quantity":0,"id":28} | item\_id=28 |
| GET /items/199 | 404 | {"detail":"item 199 not found"} | item\_id=199 |
| GET /items/626 | 404 | {"detail":"item 626 not found"} | item\_id=626 |
| GET /items/322 | 404 | {"detail":"item 322 not found"} | item\_id=322 |
| GET /items/947 | 404 | {"detail":"item 947 not found"} | item\_id=947 |
| GET /items/410 | 404 | {"detail":"item 410 not found"} | item\_id=410 |
| GET /items/829 | 404 | {"detail":"item 829 not found"} | item\_id=829 |
| GET /items/381 | 404 | {"detail":"item 381 not found"} | item\_id=381 |
| GET /items/668 | 404 | {"detail":"item 668 not found"} | item\_id=668 |
| GET /items/490 | 404 | {"detail":"item 490 not found"} | item\_id=490 |
| GET /items/238 | 404 | {"detail":"item 238 not found"} | item\_id=238 |
| GET /items/773 | 404 | {"detail":"item 773 not found"} | item\_id=773 |
| GET /items/577 | 404 | {"detail":"item 577 not found"} | item\_id=577 |
| GET /items/194 | 404 | {"detail":"item 194 not found"} | item\_id=194 |
| GET /items/64 | 200 | {"name":"乐Ã丩","description":null,"price":599.43,"quantity":0,"id":64} | item\_id=64 |
| GET /items/605 | 404 | {"detail":"item 605 not found"} | item\_id=605 |
| GET /items/365 | 404 | {"detail":"item 365 not found"} | item\_id=365 |
| GET /items/69 | 200 | {"name":"Ô万乶乙걽","description":null,"price":724.336,"quantity":880,"id":69} | item\_id=69 |
| GET /items/585 | 404 | {"detail":"item 585 not found"} | item\_id=585 |
| GET /items/54 | 200 | {"name":"𠀧Ñ","description":null,"price":196.969,"quantity":977,"id":54} | item\_id=54 |
| GET /items/899 | 404 | {"detail":"item 899 not found"} | item\_id=899 |
| GET /items/914 | 404 | {"detail":"item 914 not found"} | item\_id=914 |
| GET /items/545 | 404 | {"detail":"item 545 not found"} | item\_id=545 |
| GET /items/335 | 404 | {"detail":"item 335 not found"} | item\_id=335 |
| GET /items/385 | 404 | {"detail":"item 385 not found"} | item\_id=385 |
| GET /items/711 | 404 | {"detail":"item 711 not found"} | item\_id=711 |
| GET /items/270 | 404 | {"detail":"item 270 not found"} | item\_id=270 |
| GET /items/544 | 404 | {"detail":"item 544 not found"} | item\_id=544 |
| GET /items/977 | 404 | {"detail":"item 977 not found"} | item\_id=977 |
| GET /items/754 | 404 | {"detail":"item 754 not found"} | item\_id=754 |
| GET /items/953 | 404 | {"detail":"item 953 not found"} | item\_id=953 |
| GET /items/428 | 404 | {"detail":"item 428 not found"} | item\_id=428 |
| GET /items/701 | 404 | {"detail":"item 701 not found"} | item\_id=701 |
| GET /items/514 | 404 | {"detail":"item 514 not found"} | item\_id=514 |
| GET /items/226 | 404 | {"detail":"item 226 not found"} | item\_id=226 |
| GET /items/392 | 404 | {"detail":"item 392 not found"} | item\_id=392 |
| GET /items/738 | 404 | {"detail":"item 738 not found"} | item\_id=738 |
| GET /items/192 | 404 | {"detail":"item 192 not found"} | item\_id=192 |
| GET /items/826 | 404 | {"detail":"item 826 not found"} | item\_id=826 |
| GET /items/606 | 404 | {"detail":"item 606 not found"} | item\_id=606 |
| GET /items/99 | 200 | {"name":"並𠀱Ф乙𠁕","description":null,"price":447.77,"quantity":0,"id":99} | item\_id=99 |
| GET /items/866 | 404 | {"detail":"item 866 not found"} | item\_id=866 |
| GET /items/282 | 404 | {"detail":"item 282 not found"} | item\_id=282 |
| GET /items/703 | 404 | {"detail":"item 703 not found"} | item\_id=703 |
| GET /items/498 | 404 | {"detail":"item 498 not found"} | item\_id=498 |
| GET /items/375 | 404 | {"detail":"item 375 not found"} | item\_id=375 |
| GET /items/378 | 404 | {"detail":"item 378 not found"} | item\_id=378 |
| GET /items/536 | 404 | {"detail":"item 536 not found"} | item\_id=536 |
| GET /items/776 | 404 | {"detail":"item 776 not found"} | item\_id=776 |
| GET /items/608 | 404 | {"detail":"item 608 not found"} | item\_id=608 |
| GET /items/386 | 404 | {"detail":"item 386 not found"} | item\_id=386 |
| GET /items/237 | 404 | {"detail":"item 237 not found"} | item\_id=237 |
| GET /items/457 | 404 | {"detail":"item 457 not found"} | item\_id=457 |
| GET /items/525 | 404 | {"detail":"item 525 not found"} | item\_id=525 |
| GET /items/273 | 404 | {"detail":"item 273 not found"} | item\_id=273 |
| GET /items/177 | 404 | {"detail":"item 177 not found"} | item\_id=177 |
| GET /items/255 | 404 | {"detail":"item 255 not found"} | item\_id=255 |
| GET /items/41 | 404 | {"detail":"item 41 not found"} | item\_id=41 |
| GET /items/347 | 404 | {"detail":"item 347 not found"} | item\_id=347 |
| GET /items/298 | 404 | {"detail":"item 298 not found"} | item\_id=298 |
| GET /items/102 | 200 | {"name":"Υ갭𠀂갺み","description":"𠁉갺ई𠀣걋3𠁕","price":325.438,"quantity":0,"id":102} | item\_id=102 |
| GET /items/590 | 404 | {"detail":"item 590 not found"} | item\_id=590 |
| GET /items/322 | 404 | {"detail":"item 322 not found"} | item\_id=322 |
| GET /items/814 | 404 | {"detail":"item 814 not found"} | item\_id=814 |
| GET /items/709 | 404 | {"detail":"item 709 not found"} | item\_id=709 |
| GET /items/689 | 404 | {"detail":"item 689 not found"} | item\_id=689 |
| GET /items/504 | 404 | {"detail":"item 504 not found"} | item\_id=504 |
| GET /items/400 | 404 | {"detail":"item 400 not found"} | item\_id=400 |
| GET /items/535 | 404 | {"detail":"item 535 not found"} | item\_id=535 |
| GET /items/769 | 404 | {"detail":"item 769 not found"} | item\_id=769 |
| GET /items/505 | 404 | {"detail":"item 505 not found"} | item\_id=505 |
| GET /items/342 | 404 | {"detail":"item 342 not found"} | item\_id=342 |
| GET /items/357 | 404 | {"detail":"item 357 not found"} | item\_id=357 |
| GET /items/578 | 404 | {"detail":"item 578 not found"} | item\_id=578 |
| GET /items/633 | 404 | {"detail":"item 633 not found"} | item\_id=633 |
| GET /items/378 | 404 | {"detail":"item 378 not found"} | item\_id=378 |
| GET /items/298 | 404 | {"detail":"item 298 not found"} | item\_id=298 |
| GET /items/60 | 404 | {"detail":"item 60 not found"} | item\_id=60 |
| GET /items/444 | 404 | {"detail":"item 444 not found"} | item\_id=444 |
| GET /items/270 | 404 | {"detail":"item 270 not found"} | item\_id=270 |
| GET /items/232 | 404 | {"detail":"item 232 not found"} | item\_id=232 |
| GET /items/2 | 200 | {"name":"ÐÖ","description":null,"price":171.564,"quantity":649,"id":2} | item\_id=2 |
| GET /items/649 | 404 | {"detail":"item 649 not found"} | item\_id=649 |

## PUT /items/{item\_id}

Update Item

- **HTTP verb:** PUT
- **Path:** /items/{item\_id}
- **Operation ID:** update\_item\_items\_\_item\_id\_\_put

### Parameters

| Name | In | Type | Required | Constraints |
|---|---|---|---|---|
| item\_id | path | integer | yes | exclusive min 0 |
| (body) | application/json | object | yes |  |
| description | application/json | string (nullable) | no | max length 256 |
| name | application/json | string | yes | min length 1; max length 64 |
| price | application/json | number | yes | exclusive min 0; max 10000 |
| quantity | application/json | integer | no | min 0; max 1000 |

### Results

100 sent, 100 reported (filter: all). Status codes: 200 ×14, 404 ×86.

| Endpoint | Status | Message | Input |
|---|---|---|---|
| PUT /items/936 | 404 | {"detail":"item 936 not found"} | item\_id=936, body={"description":"Z丐ьछ걦갫7","name":"乎ЭΒぷËÆ丌э","price":29.033,"quantity":106} |
| PUT /items/346 | 404 | {"detail":"item 346 not found"} | item\_id=346, body={"name":"𠁬Ö𠀶丩","price":484.872,"quantity":816} |
| PUT /items/116 | 404 | {"detail":"item 116 not found"} | item\_id=116, body={"description":"乀Ð갦","name":"ØΜXkは𠁊Ù걙","price":906.052} |
| PUT /items/533 | 404 | {"detail":"item 533 not found"} | item\_id=533, body={"description":"いпÖ","name":"串𠀠न","price":606.107,"quantity":631} |
| PUT /items/337 | 404 | {"detail":"item 337 not found"} | item\_id=337, body={"description":"乽","name":"ةκ𠀛𠁟γङすιΜ𠁴","price":782.497,"quantity":373} |
| PUT /items/922 | 404 | {"detail":"item 922 not found"} | item\_id=922, body={"description":"A6乀乌れدऊお","name":"걒झذ갣东𠀫ЬЦ","price":279.529} |
| PUT /items/4 | 200 | {"name":"धىП","description":null,"price":543.741,"quantity":0,"id":4} | item\_id=4, body={"name":"धىП","price":543.741} |
| PUT /items/16 | 200 | {"name":"乞","description":null,"price":479.399,"quantity":63,"id":16} | item\_id=16, body={"name":"乞","price":479.399,"quantity":63} |
| PUT /items/132 | 404 | {"detail":"item 132 not found"} | item\_id=132, body={"name":"KलÏð乵乪ш9가फ","price":370.917,"quantity":240} |
| PUT /items/693 | 404 | {"detail":"item 693 not found"} | item\_id=693, body={"description":"𠀥갪れ𠁷𠀒j乘𠀀","name":"갱丐Ö丮","price":659.569} |
| PUT /items/399 | 404 | {"detail":"item 399 not found"} | item\_id=399, body={"name":"값么ऍé丸갸Λß갤","price":69.193} |
| PUT /items/723 | 404 | {"detail":"item 723 not found"} | item\_id=723, body={"description":"갊𠁍𠀦ÚюすÒपぷ","name":"ゔË𠁳","price":386.12,"quantity":499} |
| PUT /items/284 | 404 | {"detail":"item 284 not found"} | item\_id=284, body={"description":"乙","name":"ы丠𠀟Жà","price":574.56} |
| PUT /items/377 | 404 | {"detail":"item 377 not found"} | item\_id=377, body={"description":null,"name":"こ갃𠁎丱ぅΨ𠁮zÈ","price":908.945} |
| PUT /items/715 | 404 | {"detail":"item 715 not found"} | item\_id=715, body={"name":"丗丹Р갡ब걐걨","price":914.967,"quantity":59} |
| PUT /items/336 | 404 | {"detail":"item 336 not found"} | item\_id=336, body={"description":"갽","name":"乱𠀤Ýまщ1𠀷か","price":883.343} |
| PUT /items/472 | 404 | {"detail":"item 472 not found"} | item\_id=472, body={"description":"乴乑áπべÔ","name":"걝せ거乮g","price":289.666} |
| PUT /items/358 | 404 | {"detail":"item 358 not found"} | item\_id=358, body={"description":"М𠀤","name":"걇걛갈à𠁪","price":277.278,"quantity":404} |
| PUT /items/249 | 404 | {"detail":"item 249 not found"} | item\_id=249, body={"description":"ぎL𠀈æ丮","name":"乓갗𠁽でつ丆íりょô","price":155.953} |
| PUT /items/598 | 404 | {"detail":"item 598 not found"} | item\_id=598, body={"description":"ぃभ걺7","name":"Ο不丫","price":117.516} |
| PUT /items/168 | 404 | {"detail":"item 168 not found"} | item\_id=168, body={"description":"갴п","name":"R乃丘걢Щ𠁄кВM","price":377.098} |
| PUT /items/830 | 404 | {"detail":"item 830 not found"} | item\_id=830, body={"description":"もみ丿걗","name":"ऊっ𠀳𠁒ど𠁌𠀠𠁫𠁅Ç","price":506.637,"quantity":325} |
| PUT /items/393 | 404 | {"detail":"item 393 not found"} | item\_id=393, body={"description":"也乾丄𠀶вय","name":"𠀷äy𠁦𠁜","price":54.076} |
| PUT /items/915 | 404 | {"detail":"item 915 not found"} | item\_id=915, body={"name":"걚걮Áõλ걖","price":150.417} |
| PUT /items/128 | 404 | {"detail":"item 128 not found"} | item\_id=128, body={"name":"す걙𠁴𠁏𠀱갋स","price":35.606,"quantity":567} |
| PUT /items/159 | 404 | {"detail":"item 159 not found"} | item\_id=159, body={"description":"ぃdk갈걳ど𠀗jÿ","name":"Ë","price":620.433} |
| PUT /items/385 | 404 | {"detail":"item 385 not found"} | item\_id=385, body={"description":"ゅкख乼Оع걁갠ぎ걱","name":"でय","price":744.861} |
| PUT /items/594 | 404 | {"detail":"item 594 not found"} | item\_id=594, body={"name":"む𠀞","price":729.558} |
| PUT /items/901 | 404 | {"detail":"item 901 not found"} | item\_id=901, body={"description":"갈ز𠀯षखðÞД","name":"乺𠁩ばΖ𠀤","price":643.262} |
| PUT /items/767 | 404 | {"detail":"item 767 not found"} | item\_id=767, body={"name":"𠁸걢ΝÉ乺","price":496.346,"quantity":611} |
| PUT /items/85 | 200 | {"name":"Ν걘","description":null,"price":907.219,"quantity":0,"id":85} | item\_id=85, body={"name":"Ν걘","price":907.219} |
| PUT /items/548 | 404 | {"detail":"item 548 not found"} | item\_id=548, body={"description":"ί갘걱ΗÅ","name":"𠁾Ψठऔ걶Ñу","price":540.972,"quantity":937} |
| PUT /items/229 | 404 | {"detail":"item 229 not found"} | item\_id=229, body={"description":"ΠべـΚ","name":"𠀮ह","price":667.842} |
| PUT /items/824 | 404 | {"detail":"item 824 not found"} | item\_id=824, body={"description":"𠀧غऔ𠁚س","name":"दиち","price":389.321} |
| PUT /items/669 | 404 | {"detail":"item 669 not found"} | item\_id=669, body={"description":"걝𠁚丯","name":"Ýऩy걘乞","price":366.931} |
| PUT /items/636 | 404 | {"detail":"item 636 not found"} | item\_id=636, body={"name":"ञइ𠁯έΝТ3𠁃тك","price":620.365} |
| PUT /items/304 | 404 | {"detail":"item 304 not found"} | item\_id=304, body={"description":"𠀃𠁊𠁤ôW","name":"आ갩жêÛضl갺","price":473.053,"quantity":462} |
| PUT /items/634 | 404 | {"detail":"item 634 not found"} | item\_id=634, body={"description":"𠀳a","name":"Й걃𠁍Ü","price":674.499,"quantity":599} |
| PUT /items/612 | 404 | {"detail":"item 612 not found"} | item\_id=612, body={"name":"I乑","price":107.71} |
| PUT /items/515 | 404 | {"detail":"item 515 not found"} | item\_id=515, body={"name":"प걜غ갹𠀕СY乱É丩","price":766.413,"quantity":890} |
| PUT /items/605 | 404 | {"detail":"item 605 not found"} | item\_id=605, body={"name":"가","price":298.544,"quantity":676} |
| PUT /items/103 | 200 | {"name":"𠀤丈","description":null,"price":162.786,"quantity":84,"id":103} | item\_id=103, body={"description":null,"name":"𠀤丈","price":162.786,"quantity":84} |
| PUT /items/24 | 200 | {"name":"乨т乓걶И감r","description":"강갋𠁫э久てR𠀫7","price":755.711,"quantity":266,"id":24} | item\_id=24, body={"description":"강갋𠁫э久てR𠀫7","name":"乨т乓걶И감r","price":755.711,"quantity":266} |
| PUT /items/684 | 404 | {"detail":"item 684 not found"} | item\_id=684, body={"name":"丁G𠀡𠀉乽","price":881.768,"quantity":984} |
| PUT /items/178 | 404 | {"detail":"item 178 not found"} | item\_id=178, body={"description":"ا1𠀜ΰΛ𠀆V乚ग","name":"ξ","price":219.363,"quantity":715} |
| PUT /items/913 | 404 | {"detail":"item 913 not found"} | item\_id=913, body={"description":"फ𠀞","name":"जζ丳𠀋ΝZ乢ïæػ","price":689.321} |
| PUT /items/869 | 404 | {"detail":"item 869 not found"} | item\_id=869, body={"name":"乓𠀲걹ङ丿乆걮Н","price":930.65} |
| PUT /items/461 | 404 | {"detail":"item 461 not found"} | item\_id=461, body={"name":"れхゐحs","price":991.162,"quantity":945} |
| PUT /items/711 | 404 | {"detail":"item 711 not found"} | item\_id=711, body={"description":"かδ𠀜𠁦걬갲","name":"g乣𠀶丕घةËさΑ","price":251.41} |
| PUT /items/224 | 404 | {"detail":"item 224 not found"} | item\_id=224, body={"description":"uг丽东乲た","name":"k","price":859.497,"quantity":551} |
| PUT /items/943 | 404 | {"detail":"item 943 not found"} | item\_id=943, body={"name":"𠁀ん걍У걪К걗갱걨갭","price":658.955,"quantity":550} |
| PUT /items/270 | 404 | {"detail":"item 270 not found"} | item\_id=270, body={"description":"걽","name":"걜ω𠁞X𠁙乃","price":305.9,"quantity":197} |
| PUT /items/327 | 404 | {"detail":"item 327 not found"} | item\_id=327, body={"name":"𠀱Ô갛ط갊丝丛","price":433.435} |
| PUT /items/538 | 404 | {"detail":"item 538 not found"} | item\_id=538, body={"name":"いれÁΩ𠁓Е걆ふC","price":899.553} |
| PUT /items/795 | 404 | {"detail":"item 795 not found"} | item\_id=795, body={"name":"乴𠁫फ","price":95.177,"quantity":787} |
| PUT /items/204 | 404 | {"detail":"item 204 not found"} | item\_id=204, body={"description":"ざや乪강𠁘","name":"ぺऌफгП","price":907.516} |
| PUT /items/880 | 404 | {"detail":"item 880 not found"} | item\_id=880, body={"description":"갊乥ぐ","name":"갭𠀪अ","price":217.959} |
| PUT /items/139 | 200 | {"name":"Ü三丛u丝w","description":"𠁪걨Cぐй个С","price":657.198,"quantity":747,"id":139} | item\_id=139, body={"description":"𠁪걨Cぐй个С","name":"Ü三丛u丝w","price":657.198,"quantity":747} |
| PUT /items/283 | 404 | {"detail":"item 283 not found"} | item\_id=283, body={"description":"걖Υ𠀙𠁴もn갷ض丼갷","name":"R上Ν1걐丨Ûじ","price":603.262,"quantity":499} |
| PUT /items/687 | 404 | {"detail":"item 687 not found"} | item\_id=687, body={"description":"p걽","name":"义ÛCΫýψ丧ت","price":709.466} |
| PUT /items/767 | 404 | {"detail":"item 767 not found"} | item\_id=767, body={"description":"さ갽乩걥걏ई乫ふЧ","name":"ढ两𠀋","price":385.353,"quantity":282} |
| PUT /items/865 | 404 | {"detail":"item 865 not found"} | item\_id=865, body={"description":"おO𠀩ぞ","name":"乆𠁒丂ش","price":99.171} |
| PUT /items/812 | 404 | {"detail":"item 812 not found"} | item\_id=812, body={"name":"व乮","price":379.023} |
| PUT /items/696 | 404 | {"detail":"item 696 not found"} | item\_id=696, body={"name":"갸や之𠀜ぼ걞","price":624.399} |
| PUT /items/208 | 404 | {"detail":"item 208 not found"} | item\_id=208, body={"name":"lऐ강æぱ𠁶갰د𠁉ك","price":336.641} |
| PUT /items/189 | 404 | {"detail":"item 189 not found"} | item\_id=189, body={"description":"갨ゐ𠁎Õफ","name":"𠁗き东З1υ𠀝mς","price":520.001} |
| PUT /items/565 | 404 | {"detail":"item 565 not found"} | item\_id=565, body={"name":"갩ओ𠁄乽𠀋य","price":955.683,"quantity":840} |
| PUT /items/10 | 200 | {"name":"चб","description":null,"price":546.639,"quantity":968,"id":10} | item\_id=10, body={"name":"चб","price":546.639,"quantity":968} |
| PUT /items/320 | 404 | {"detail":"item 320 not found"} | item\_id=320, body={"description":null,"name":"र객R","price":91.241,"quantity":720} |
| PUT /items/182 | 404 | {"detail":"item 182 not found"} | item\_id=182, body={"name":"えΧ九","price":95.178,"quantity":133} |
| PUT /items/789 | 404 | {"detail":"item 789 not found"} | item\_id=789, body={"description":"ج","name":"fО","price":762.892,"quantity":495} |
| PUT /items/103 | 200 | {"name":"专丅两इ걺갵प걭","description":null,"price":39.841,"quantity":397,"id":103} | item\_id=103, body={"name":"专丅两इ걺갵प걭","price":39.841,"quantity":397} |
| PUT /items/205 | 404 | {"detail":"item 205 not found"} | item\_id=205, body={"name":"Γに𠀨lз감不不갧𠀗","price":81.303,"quantity":681} |
| PUT /items/539 | 404 | {"detail":"item 539 not found"} | item\_id=539, body={"description":"乫","name":"ت𠁃𠀉","price":494.725,"quantity":497} |
| PUT /items/599 | 404 | {"detail":"item 599 not found"} | item\_id=599, body={"name":"갉ΚΩ걼𠀕𠁗","price":193.638} |
| PUT /items/533 | 404 | {"detail":"item 533 not found"} | item\_id=533, body={"description":"8み丗乏ç𠀝","name":"がуâ간𠀬ζ","price":450.197} |
| PUT /items/648 | 404 | {"detail":"item 648 not found"} | item\_id=648, body={"description":"Ã걩Ю거举","name":"𠁄λ𠀻","price":962.779} |
| PUT /items/78 | 200 | {"name":"九𠀣شþ丿乒","description":null,"price":943.225,"quantity":862,"id":78} | item\_id=78, body={"name":"九𠀣شþ丿乒","price":943.225,"quantity":862} |
| PUT /items/715 | 404 | {"detail":"item 715 not found"} | item\_id=715, body={"name":"감м𠁃ΡE","price":276.18,"quantity":273} |
| PUT /items/809 | 404 | {"detail":"item 809 not found"} | item\_id=809, body={"name":"ôぺ1𠁟丢Шくड걜中","price":567.067,"quantity":779} |
| PUT /items/992 | 404 | {"detail":"item 992 not found"} | item\_id=992, body={"name":"ずÄ乀छ","price":199.266,"quantity":731} |
| PUT /items/314 | 404 | {"detail":"item 314 not found"} | item\_id=314, body={"name":"ß","price":234.022} |
| PUT /items/356 | 404 | {"detail":"item 356 not found"} | item\_id=356, body={"description":"갏걩ςА","name":"걎𠁟ضङر","price":943.597,"quantity":798} |
| PUT /items/76 | 200 | {"name":"𠀅","description":null,"price":644.699,"quantity":0,"id":76} | item\_id=76, body={"name":"𠀅","price":644.699} |
| PUT /items/672 | 404 | {"detail":"item 672 not found"} | item\_id=672, body={"description":"0","name":"zट","price":336.45} |
| PUT /items/861 | 404 | {"detail":"item 861 not found"} | item\_id=861, body={"description":"й걇Й","name":"Χγ걷","price":959.791,"quantity":329} |
| PUT /items/200 | 404 | {"detail":"item 200 not found"} | item\_id=200, body={"name":"pGëचcतゑ𠀐и","price":750.474} |
| PUT /items/800 | 404 | {"detail":"item 800 not found"} | item\_id=800, body={"description":null,"name":"П𠁝丐ぞゕदをで","price":520.194} |
| PUT /items/951 | 404 | {"detail":"item 951 not found"} | item\_id=951, body={"name":"く不ぶछ걕𠀗","price":993.372,"quantity":986} |
| PUT /items/149 | 200 | {"name":"𠁮ي乌𠁱𠁘аË丧έ乪","description":"Е걛よ갹れC","price":18.065,"quantity":0,"id":149} | item\_id=149, body={"description":"Е걛よ갹れC","name":"𠁮ي乌𠁱𠁘аË丧έ乪","price":18.065} |
| PUT /items/721 | 404 | {"detail":"item 721 not found"} | item\_id=721, body={"name":"갃𠁞","price":65.793} |
| PUT /items/869 | 404 | {"detail":"item 869 not found"} | item\_id=869, body={"description":"𠀱ΰÿ𠁱걹ы𠁢Φ𠀼j","name":"ви7ú걋","price":292.552,"quantity":662} |
| PUT /items/468 | 404 | {"detail":"item 468 not found"} | item\_id=468, body={"name":"Ъ갑","price":90.069} |
| PUT /items/410 | 404 | {"detail":"item 410 not found"} | item\_id=410, body={"name":"개갣丸j걹𠁨õ간𠀕","price":15.796,"quantity":146} |
| PUT /items/991 | 404 | {"detail":"item 991 not found"} | item\_id=991, body={"description":"Ñì𠁃갂ぢ𠀔三ば","name":"𠀽𠁟سъ","price":379.674,"quantity":892} |
| PUT /items/943 | 404 | {"detail":"item 943 not found"} | item\_id=943, body={"name":"𠁢𠁽갤갵𠁖","price":930.626} |
| PUT /items/127 | 200 | {"name":"ПÁH𠀆걮𠁛á𠀌Ο","description":null,"price":245.923,"quantity":0,"id":127} | item\_id=127, body={"name":"ПÁH𠀆걮𠁛á𠀌Ο","price":245.923} |
| PUT /items/149 | 200 | {"name":"き갼갳õËò举","description":"乘","price":837.339,"quantity":703,"id":149} | item\_id=149, body={"description":"乘","name":"き갼갳õËò举","price":837.339,"quantity":703} |
| PUT /items/367 | 404 | {"detail":"item 367 not found"} | item\_id=367, body={"name":"乍𠁒ûç乡걧丄𠁲ςつ","price":450.695,"quantity":855} |
| PUT /items/94 | 200 | {"name":"一Ø","description":null,"price":593.403,"quantity":473,"id":94} | item\_id=94, body={"name":"一Ø","price":593.403,"quantity":473} |

## DELETE /items/{item\_id}

Delete Item

- **HTTP verb:** DELETE
- **Path:** /items/{item\_id}
- **Operation ID:** delete\_item\_items\_\_item\_id\_\_delete

### Parameters

| Name | In | Type | Required | Constraints |
|---|---|---|---|---|
| item\_id | path | integer | yes | exclusive min 0 |

### Results

100 sent, 100 reported (filter: all). Status codes: 204 ×16, 404 ×84.

| Endpoint | Status | Message | Input |
|---|---|---|---|
| DELETE /items/827 | 404 | {"detail":"item 827 not found"} | item\_id=827 |
| DELETE /items/688 | 404 | {"detail":"item 688 not found"} | item\_id=688 |
| DELETE /items/666 | 404 | {"detail":"item 666 not found"} | item\_id=666 |
| DELETE /items/735 | 404 | {"detail":"item 735 not found"} | item\_id=735 |
| DELETE /items/998 | 404 | {"detail":"item 998 not found"} | item\_id=998 |
| DELETE /items/492 | 404 | {"detail":"item 492 not found"} | item\_id=492 |
| DELETE /items/708 | 404 | {"detail":"item 708 not found"} | item\_id=708 |
| DELETE /items/438 | 404 | {"detail":"item 438 not found"} | item\_id=438 |
| DELETE /items/663 | 404 | {"detail":"item 663 not found"} | item\_id=663 |
| DELETE /items/77 | 204 | No Content | item\_id=77 |
| DELETE /items/651 | 404 | {"detail":"item 651 not found"} | item\_id=651 |
| DELETE /items/193 | 404 | {"detail":"item 193 not found"} | item\_id=193 |
| DELETE /items/445 | 404 | {"detail":"item 445 not found"} | item\_id=445 |
| DELETE /items/698 | 404 | {"detail":"item 698 not found"} | item\_id=698 |
| DELETE /items/490 | 404 | {"detail":"item 490 not found"} | item\_id=490 |
| DELETE /items/65 | 204 | No Content | item\_id=65 |
| DELETE /items/653 | 404 | {"detail":"item 653 not found"} | item\_id=653 |
| DELETE /items/198 | 404 | {"detail":"item 198 not found"} | item\_id=198 |
| DELETE /items/296 | 404 | {"detail":"item 296 not found"} | item\_id=296 |
| DELETE /items/301 | 404 | {"detail":"item 301 not found"} | item\_id=301 |
| DELETE /items/280 | 404 | {"detail":"item 280 not found"} | item\_id=280 |
| DELETE /items/842 | 404 | {"detail":"item 842 not found"} | item\_id=842 |
| DELETE /items/668 | 404 | {"detail":"item 668 not found"} | item\_id=668 |
| DELETE /items/935 | 404 | {"detail":"item 935 not found"} | item\_id=935 |
| DELETE /items/60 | 204 | No Content | item\_id=60 |
| DELETE /items/797 | 404 | {"detail":"item 797 not found"} | item\_id=797 |
| DELETE /items/685 | 404 | {"detail":"item 685 not found"} | item\_id=685 |
| DELETE /items/554 | 404 | {"detail":"item 554 not found"} | item\_id=554 |
| DELETE /items/23 | 204 | No Content | item\_id=23 |
| DELETE /items/610 | 404 | {"detail":"item 610 not found"} | item\_id=610 |
| DELETE /items/756 | 404 | {"detail":"item 756 not found"} | item\_id=756 |
| DELETE /items/671 | 404 | {"detail":"item 671 not found"} | item\_id=671 |
| DELETE /items/134 | 404 | {"detail":"item 134 not found"} | item\_id=134 |
| DELETE /items/793 | 404 | {"detail":"item 793 not found"} | item\_id=793 |
| DELETE /items/137 | 404 | {"detail":"item 137 not found"} | item\_id=137 |
| DELETE /items/5 | 204 | No Content | item\_id=5 |
| DELETE /items/123 | 204 | No Content | item\_id=123 |
| DELETE /items/215 | 404 | {"detail":"item 215 not found"} | item\_id=215 |
| DELETE /items/298 | 404 | {"detail":"item 298 not found"} | item\_id=298 |
| DELETE /items/335 | 404 | {"detail":"item 335 not found"} | item\_id=335 |
| DELETE /items/432 | 404 | {"detail":"item 432 not found"} | item\_id=432 |
| DELETE /items/481 | 404 | {"detail":"item 481 not found"} | item\_id=481 |
| DELETE /items/7 | 204 | No Content | item\_id=7 |
| DELETE /items/719 | 404 | {"detail":"item 719 not found"} | item\_id=719 |
| DELETE /items/964 | 404 | {"detail":"item 964 not found"} | item\_id=964 |
| DELETE /items/59 | 204 | No Content | item\_id=59 |
| DELETE /items/218 | 404 | {"detail":"item 218 not found"} | item\_id=218 |
| DELETE /items/319 | 404 | {"detail":"item 319 not found"} | item\_id=319 |
| DELETE /items/717 | 404 | {"detail":"item 717 not found"} | item\_id=717 |
| DELETE /items/455 | 404 | {"detail":"item 455 not found"} | item\_id=455 |
| DELETE /items/831 | 404 | {"detail":"item 831 not found"} | item\_id=831 |
| DELETE /items/146 | 204 | No Content | item\_id=146 |
| DELETE /items/877 | 404 | {"detail":"item 877 not found"} | item\_id=877 |
| DELETE /items/264 | 404 | {"detail":"item 264 not found"} | item\_id=264 |
| DELETE /items/966 | 404 | {"detail":"item 966 not found"} | item\_id=966 |
| DELETE /items/301 | 404 | {"detail":"item 301 not found"} | item\_id=301 |
| DELETE /items/105 | 204 | No Content | item\_id=105 |
| DELETE /items/945 | 404 | {"detail":"item 945 not found"} | item\_id=945 |
| DELETE /items/445 | 404 | {"detail":"item 445 not found"} | item\_id=445 |
| DELETE /items/789 | 404 | {"detail":"item 789 not found"} | item\_id=789 |
| DELETE /items/628 | 404 | {"detail":"item 628 not found"} | item\_id=628 |
| DELETE /items/735 | 404 | {"detail":"item 735 not found"} | item\_id=735 |
| DELETE /items/26 | 204 | No Content | item\_id=26 |
| DELETE /items/601 | 404 | {"detail":"item 601 not found"} | item\_id=601 |
| DELETE /items/969 | 404 | {"detail":"item 969 not found"} | item\_id=969 |
| DELETE /items/521 | 404 | {"detail":"item 521 not found"} | item\_id=521 |
| DELETE /items/572 | 404 | {"detail":"item 572 not found"} | item\_id=572 |
| DELETE /items/474 | 404 | {"detail":"item 474 not found"} | item\_id=474 |
| DELETE /items/848 | 404 | {"detail":"item 848 not found"} | item\_id=848 |
| DELETE /items/852 | 404 | {"detail":"item 852 not found"} | item\_id=852 |
| DELETE /items/919 | 404 | {"detail":"item 919 not found"} | item\_id=919 |
| DELETE /items/767 | 404 | {"detail":"item 767 not found"} | item\_id=767 |
| DELETE /items/26 | 404 | {"detail":"item 26 not found"} | item\_id=26 |
| DELETE /items/429 | 404 | {"detail":"item 429 not found"} | item\_id=429 |
| DELETE /items/291 | 404 | {"detail":"item 291 not found"} | item\_id=291 |
| DELETE /items/993 | 404 | {"detail":"item 993 not found"} | item\_id=993 |
| DELETE /items/958 | 404 | {"detail":"item 958 not found"} | item\_id=958 |
| DELETE /items/639 | 404 | {"detail":"item 639 not found"} | item\_id=639 |
| DELETE /items/144 | 204 | No Content | item\_id=144 |
| DELETE /items/609 | 404 | {"detail":"item 609 not found"} | item\_id=609 |
| DELETE /items/481 | 404 | {"detail":"item 481 not found"} | item\_id=481 |
| DELETE /items/916 | 404 | {"detail":"item 916 not found"} | item\_id=916 |
| DELETE /items/943 | 404 | {"detail":"item 943 not found"} | item\_id=943 |
| DELETE /items/728 | 404 | {"detail":"item 728 not found"} | item\_id=728 |
| DELETE /items/276 | 404 | {"detail":"item 276 not found"} | item\_id=276 |
| DELETE /items/408 | 404 | {"detail":"item 408 not found"} | item\_id=408 |
| DELETE /items/433 | 404 | {"detail":"item 433 not found"} | item\_id=433 |
| DELETE /items/32 | 204 | No Content | item\_id=32 |
| DELETE /items/420 | 404 | {"detail":"item 420 not found"} | item\_id=420 |
| DELETE /items/95 | 204 | No Content | item\_id=95 |
| DELETE /items/343 | 404 | {"detail":"item 343 not found"} | item\_id=343 |
| DELETE /items/922 | 404 | {"detail":"item 922 not found"} | item\_id=922 |
| DELETE /items/101 | 204 | No Content | item\_id=101 |
| DELETE /items/77 | 404 | {"detail":"item 77 not found"} | item\_id=77 |
| DELETE /items/98 | 204 | No Content | item\_id=98 |
| DELETE /items/518 | 404 | {"detail":"item 518 not found"} | item\_id=518 |
| DELETE /items/642 | 404 | {"detail":"item 642 not found"} | item\_id=642 |
| DELETE /items/249 | 404 | {"detail":"item 249 not found"} | item\_id=249 |
| DELETE /items/743 | 404 | {"detail":"item 743 not found"} | item\_id=743 |
| DELETE /items/678 | 404 | {"detail":"item 678 not found"} | item\_id=678 |

