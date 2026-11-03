# Fuzzy report

| | |
|---|---|
| **Target** | http://localhost:8000 |
| **Spec** | ./examples/crud/artifacts/openapi.json (Fuzzy CRUD example 1.0.0) |
| **Dictionary** | none (random values) |
| **Random bounds** | <ul><li>string: {min: 1, max: 10}</li><li>int: {min: 0, max: 1000}</li><li>float: {min: 0, max: 1000}</li></ul> |
| **Charset** | unicode |
| **Status filter** | all |
| **Seed** | 4121700419951235171 |
| **Started** | 2026-11-03T02:26:27+01:00 |
| **Duration** | 15ms |
| **Requests sent** | 500 |
| **Results reported** | 500 |

## Summary

| Endpoint | Sent | Reported | Status codes |
|---|---|---|---|
| GET /items | 100 | 100 | ERR ×100 |
| POST /items | 100 | 100 | ERR ×100 |
| GET /items/{item\_id} | 100 | 100 | ERR ×100 |
| PUT /items/{item\_id} | 100 | 100 | ERR ×100 |
| DELETE /items/{item\_id} | 100 | 100 | ERR ×100 |

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

100 sent, 100 reported (filter: all). Status codes: ERR ×100.

| Endpoint | Status | Message | Input |
|---|---|---|---|
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=237 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=624 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=78, offset=260 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=57 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=2 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=56, offset=780 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=158 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=513 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=60, offset=454 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=88, offset=978 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=48, offset=969 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=64 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=52 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=76, offset=385 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=797 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=50 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=555 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=97, offset=712 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=200 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=86 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=83 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=30, offset=194 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=349 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=15 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=45 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=1 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=96, offset=193 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=87 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=98, offset=828 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=71 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=57 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=36, offset=806 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=274 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=24, offset=950 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=12 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=706 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=831 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=715 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=74 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=62 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=38 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=84, offset=121 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=188 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=7 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=369 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=55, offset=712 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=44 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=50 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=547 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=72, offset=166 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=69 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=74, offset=232 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=60 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=4, offset=669 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=532 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=28, offset=432 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=89 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=27, offset=581 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=75, offset=991 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=178 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=933 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused |  |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=18 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=388 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=27, offset=296 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=351 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | limit=10, offset=1000 |
| GET /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | offset=40 |

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

100 sent, 100 reported (filter: all). Status codes: ERR ×100.

| Endpoint | Status | Message | Input |
|---|---|---|---|
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"ऊ","price":346.317,"quantity":950} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"丂श乫걁么","name":"У","price":263.499,"quantity":593} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"ऒट𠁩걸ÿ","price":802.975} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"ぅЛ","name":"لнऴ𠀐êh","price":454.608} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"𠀶ΰæس","name":"걟ы9р𠀡Ñ乖Π业","price":718.738,"quantity":627} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":null,"name":"अ𠀹걕âほ乱Tゐ","price":122.243,"quantity":341} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"乖가𠀌ã七𠀜乲δ걯κ","name":"丐갆ऑâ並乚","price":712.397,"quantity":235} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"갌갠乭É갫Η걙","price":118.152,"quantity":448} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":null,"name":"Β","price":308.973,"quantity":351} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"𠁅ΥはНВ0걻买","price":4.132,"quantity":268} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"갼ρおh","name":"ससです","price":824.958,"quantity":360} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"せÞ𠁧","name":"ゎλðहмeÔ걓","price":972.291,"quantity":253} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"Ï中J乑丫δ𠁗","price":834.484,"quantity":453} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"ω","name":"ч걘нEÄέも","price":143.56,"quantity":536} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"ά","name":"걟丟𠀨ض𠀠걚","price":2.613,"quantity":838} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"ऋ걧𠀊乖け","price":558.525,"quantity":229} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"ょv丧Éळe乊A𠀽ष","price":93.063,"quantity":801} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"ت","name":"걓ЧΓ主갋mΖê𠁣","price":754.419,"quantity":477} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"È갬î갣걉乡ध丶","price":146.465,"quantity":669} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"걁걉Ш𠀥अ","price":936.14,"quantity":120} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"Еά乂乧갫","price":960.892,"quantity":211} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"لý0ऍ걼さ걜ؿ𠁠𠁵","name":"乢乌강가Ð乸𠀐𠀏Ó걐","price":901.617,"quantity":192} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"इo","name":"𠀶a𠀰vبщ","price":391.618} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"ì𠁬𠁠Ýض习걣𠀟","price":253.908} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"к갟𠀥乡そ","price":849.722,"quantity":770} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"𠁴Ä𠁺Ψ丕갊","price":388.643} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"𠀳乐","name":"ゃऍÜ갠","price":738.783} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"걆ëjسШν","name":"걪Ùoゅ𠁒","price":270.535} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"걹𠀼개","price":637.766} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"𠁵갘丫","price":95.289} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"갣ぴعÛmजеTψ","price":766.574} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"уऒ𠀹zã乫临","price":925.183} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"て乧झ𠀬乡","price":410.158} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"みΣش中걝","name":"êね𠀁","price":368.05,"quantity":226} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"ح걳","name":"걸ぐれそ걓Ë","price":695.482,"quantity":42} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"𠁎î𠁥丬","price":53.23,"quantity":552} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"갃し𠀑ख𠁲t1丬丰","name":"ÅÛW2丌","price":102.565} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"丛乘갉ねぱ","price":449.861} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"ऍ両𠁭𠀝","price":955.685,"quantity":785} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"こ3𠀘𠁨乽Ö걐ì𠁙","price":320.962,"quantity":288} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"𠀖В乁Tぇ𠁂づ갻𠀲е","price":519.224,"quantity":152} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"ぎ乣𠁕丟乖ôЮع","price":593.259} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"эぃ걾갢𠀻","name":"oÝþぞ中𠀇乄걪下ή","price":528.607,"quantity":392} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":null,"name":"ब갋乔Ú乽丬乩𠀨ν","price":100.638,"quantity":118} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"ぢwκ","price":484.645} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"Ф갞","price":113.392} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"𠁪","price":809.132} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"ñ걟Í𠁽걕","price":714.976,"quantity":871} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"갳Яf乸ί","price":619.425} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"乒严걖ゐ2걩","name":"ìëлぉ걻ळ","price":760.106,"quantity":514} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"乁È丷н𠁦ز","name":"갃έ걊","price":202.551} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"hق𠁵𠁮","name":"üςw乷ßく𠀚乸Π","price":625.677,"quantity":135} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"么갺걜Ф걢갳갺","price":93.361,"quantity":571} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"下KゖΙ","price":22.421} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"लぐД갾गز𠁝","name":"ठ걹Дしだयÿ","price":623.714,"quantity":186} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"なí","name":"𠀐ओo갇Υठ1け","price":546.769,"quantity":975} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"걃갫","name":"इぃ乓𠁘っぇ𠁺","price":332.52} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"걼𠀳ã갈与걉걞õ걬が","price":334.456} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"x걬与Ε𠀋え","price":195.312} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"ΰ丕걠걝걁","name":"𠁱丽ゐ갘DäХ걱","price":227.425,"quantity":97} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"مReゎ丼Çظ객","price":283.645,"quantity":425} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"ø걉","name":"ذo𠀼","price":799.856} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"η","name":"す𠀼3","price":321.731,"quantity":303} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"𠀶","name":"П","price":855.036,"quantity":172} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"ल갈걨ؽεを","name":"э감𠁿","price":677.695,"quantity":209} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"乊乒건ΓΞ乁𠁕","name":"ぜ丝걾丘ق乀","price":505.364} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"东𠁷业乕𠁁乤𠀶𠁶","price":277.654} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"ع乨갘𠀃べず갚걸","name":"Иぜ𠁏ز乤갴Þ","price":572.14,"quantity":449} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"í书걿걍ぬऍあ갰𠁰","name":"ΓÈञПüذч","price":145.278,"quantity":893} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"や丳𠁄𠁪丯め","name":"Øぶ𠀜","price":882.693} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"걣갿𠁾걨为걜乢れ乮","price":304.864,"quantity":381} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"世","name":"𠁥6𠀄ε","price":636.329} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"åزढ𠀻ؾ","name":"𠁓𠀟د걯丳乹JÎ","price":310.183} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"𠁖","price":976.452,"quantity":170} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"νПUп𠁈わ","price":899.794} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"乢𠁔り𠁓","price":647.474,"quantity":191} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"𠁈","price":193.603,"quantity":486} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"ьط𠁎","price":248.691} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"갭𠀙乶걤ã갞य𠀅乤M","price":920.028} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"𠀎l갸ゆ","price":984.292} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"りE","price":242.543} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"𠁅q乙z𠁜걾てνю之","name":"𠀿갈갡","price":835.183,"quantity":214} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"ПρчπЯいयБ丼","name":"걩Çн걑りみÉ갍","price":957.342,"quantity":780} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"άjЧд𠀻𠀪おऒ걣𠀱","price":366.186} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"yふ걥わÀい걘𠁗êΞ","name":"ई","price":356.836} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"न","name":"ぞÙ丯せюΪ","price":416.18,"quantity":859} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"𠁯𠀑갍","name":"bΙ갌Г걘ΰ𠀢","price":877.217} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"σE丝み","price":243.026} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"걹𠁬К걟rк丯","price":838.554} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"乺w갳𠁋σईةむ걙ऌ","name":"丳걨V万𠀯ش𠀔乍","price":840.087} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"Σ乫4걑丆𠁩걚","price":627.614,"quantity":498} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"걅걬ठせ𠀰í","name":"फGをеa","price":993.027,"quantity":513} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"κöز𠁨すu乂𠁃갯Щ","name":"ぽけЯ𠁀𠁟Ι𠁟ゃ","price":178.647,"quantity":178} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"갃Юî","name":"𠁧ぃ乍зتऎ丞","price":332.373,"quantity":807} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"ぞ业갔ひ","name":"S𠀌𠀖","price":469.742} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"М","name":"ΗてЗ","price":96.53,"quantity":504} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"description":"乽ظ","name":"乯𠀄ل乆上","price":422.974} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"х𠀛σμ","price":725.862} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"乲ऎ","price":787.864,"quantity":979} |
| POST /items | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | body={"name":"む갆Ϋ𠁄о","price":479.203} |

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

100 sent, 100 reported (filter: all). Status codes: ERR ×100.

| Endpoint | Status | Message | Input |
|---|---|---|---|
| GET /items/654 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=654 |
| GET /items/151 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=151 |
| GET /items/547 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=547 |
| GET /items/449 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=449 |
| GET /items/437 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=437 |
| GET /items/452 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=452 |
| GET /items/171 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=171 |
| GET /items/387 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=387 |
| GET /items/561 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=561 |
| GET /items/61 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=61 |
| GET /items/153 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=153 |
| GET /items/618 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=618 |
| GET /items/153 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=153 |
| GET /items/698 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=698 |
| GET /items/231 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=231 |
| GET /items/750 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=750 |
| GET /items/847 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=847 |
| GET /items/440 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=440 |
| GET /items/975 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=975 |
| GET /items/87 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=87 |
| GET /items/67 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=67 |
| GET /items/912 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=912 |
| GET /items/671 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=671 |
| GET /items/926 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=926 |
| GET /items/430 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=430 |
| GET /items/260 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=260 |
| GET /items/632 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=632 |
| GET /items/728 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=728 |
| GET /items/693 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=693 |
| GET /items/237 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=237 |
| GET /items/120 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=120 |
| GET /items/343 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=343 |
| GET /items/648 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=648 |
| GET /items/744 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=744 |
| GET /items/714 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=714 |
| GET /items/489 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=489 |
| GET /items/774 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=774 |
| GET /items/901 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=901 |
| GET /items/666 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=666 |
| GET /items/59 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=59 |
| GET /items/539 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=539 |
| GET /items/488 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=488 |
| GET /items/128 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=128 |
| GET /items/795 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=795 |
| GET /items/263 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=263 |
| GET /items/192 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=192 |
| GET /items/710 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=710 |
| GET /items/161 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=161 |
| GET /items/559 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=559 |
| GET /items/251 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=251 |
| GET /items/140 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=140 |
| GET /items/155 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=155 |
| GET /items/622 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=622 |
| GET /items/371 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=371 |
| GET /items/557 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=557 |
| GET /items/978 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=978 |
| GET /items/510 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=510 |
| GET /items/845 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=845 |
| GET /items/593 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=593 |
| GET /items/683 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=683 |
| GET /items/199 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=199 |
| GET /items/464 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=464 |
| GET /items/250 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=250 |
| GET /items/759 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=759 |
| GET /items/413 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=413 |
| GET /items/239 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=239 |
| GET /items/841 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=841 |
| GET /items/299 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=299 |
| GET /items/984 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=984 |
| GET /items/877 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=877 |
| GET /items/471 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=471 |
| GET /items/547 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=547 |
| GET /items/245 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=245 |
| GET /items/544 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=544 |
| GET /items/365 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=365 |
| GET /items/792 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=792 |
| GET /items/718 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=718 |
| GET /items/814 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=814 |
| GET /items/975 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=975 |
| GET /items/448 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=448 |
| GET /items/721 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=721 |
| GET /items/1000 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=1000 |
| GET /items/818 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=818 |
| GET /items/583 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=583 |
| GET /items/587 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=587 |
| GET /items/814 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=814 |
| GET /items/489 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=489 |
| GET /items/684 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=684 |
| GET /items/887 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=887 |
| GET /items/403 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=403 |
| GET /items/623 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=623 |
| GET /items/171 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=171 |
| GET /items/515 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=515 |
| GET /items/896 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=896 |
| GET /items/551 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=551 |
| GET /items/116 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=116 |
| GET /items/803 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=803 |
| GET /items/220 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=220 |
| GET /items/560 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=560 |
| GET /items/974 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=974 |

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

100 sent, 100 reported (filter: all). Status codes: ERR ×100.

| Endpoint | Status | Message | Input |
|---|---|---|---|
| PUT /items/166 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=166, body={"name":"乘हö","price":235.987,"quantity":501} |
| PUT /items/674 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=674, body={"name":"丣ï","price":480.553} |
| PUT /items/519 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=519, body={"description":"υΜ갲îさ","name":"Θ乪ةそяफ𠁘𠁻","price":954.411,"quantity":258} |
| PUT /items/836 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=836, body={"description":"걣á갨4Г为갧","name":"丫𠁱𠀓ؼ乺ゎ乿","price":48.42} |
| PUT /items/296 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=296, body={"name":"ぴकSゐе걅ý","price":199.553,"quantity":390} |
| PUT /items/650 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=650, body={"description":null,"name":"丼","price":857.621} |
| PUT /items/660 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=660, body={"name":"х𠀡갦उ𠀿𠀹갾갌","price":107.596} |
| PUT /items/950 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=950, body={"description":"なはल上ج𠁧","name":"س𠀚𠁅걄걻Α乓𠀸ぐ𠁿","price":512.413,"quantity":572} |
| PUT /items/685 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=685, body={"description":"O갗","name":"걝𠀸𠁴𠁪也","price":300.097} |
| PUT /items/441 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=441, body={"name":"𠁳ょぷس","price":726.333} |
| PUT /items/49 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=49, body={"name":"ご𠁊","price":432.224,"quantity":635} |
| PUT /items/23 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=23, body={"name":"ςя갤","price":489.805} |
| PUT /items/454 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=454, body={"description":"並ЬとP両Ь乨","name":"乷不갳걒乂ゖÐ乬丯प","price":283.668,"quantity":644} |
| PUT /items/607 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=607, body={"description":"ゐ乜ËまYΩ𠀅","name":"乂たs𠁒𠁼","price":520.783} |
| PUT /items/463 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=463, body={"description":"É걡Α","name":"ê","price":189.974,"quantity":736} |
| PUT /items/804 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=804, body={"description":"习乜ऊ","name":"ぐΦ","price":81.969,"quantity":94} |
| PUT /items/759 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=759, body={"description":"𠀭𠁜간ه一乯乾𠁅φ","name":"걒Ч𠀸خΘ걕め갔य걎","price":80.442,"quantity":772} |
| PUT /items/688 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=688, body={"name":"걟乻SΦ主丑","price":834.005,"quantity":89} |
| PUT /items/854 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=854, body={"description":"शëてоуن临","name":"Λ𠁃м𠁐","price":596.941} |
| PUT /items/927 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=927, body={"description":"ô𠀛世乔びπ𠁡","name":"z乖å𠁍也乘갰丑𠁼ه","price":980.546,"quantity":187} |
| PUT /items/38 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=38, body={"name":"Lä乔걷𠀴ゖ乌t","price":616.53,"quantity":320} |
| PUT /items/141 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=141, body={"description":"갻U丨ñΒكàऍ","name":"𠁶乶","price":5.4,"quantity":913} |
| PUT /items/502 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=502, body={"name":"𠀒걳ЩせC𠁓","price":888.417} |
| PUT /items/920 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=920, body={"description":"中举さط4下íч3C","name":"𠀏छ乐त걋","price":409.428} |
| PUT /items/442 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=442, body={"description":null,"name":"강Âè𠁫一ö","price":339.058,"quantity":430} |
| PUT /items/673 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=673, body={"description":"𠁠𠀻い乿ا久tΦ","name":"b갉च𠀞","price":625.736} |
| PUT /items/755 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=755, body={"description":"Qـ","name":"𠁳","price":837.771,"quantity":855} |
| PUT /items/758 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=758, body={"name":"𠀬ぼळ𠁑чゅ","price":81.479} |
| PUT /items/885 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=885, body={"name":"नखòa걠乽𠁉乌𠀹","price":527.297} |
| PUT /items/712 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=712, body={"description":"ö","name":"り걇ज2𠁏arいBÈ","price":380.633} |
| PUT /items/643 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=643, body={"description":"乨𠁫むせطлξ丄く丑","name":"きआ","price":811.68,"quantity":907} |
| PUT /items/926 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=926, body={"description":"갍ぎ","name":"н","price":905.189,"quantity":663} |
| PUT /items/918 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=918, body={"name":"なはぞ乹0ίνऍ갂Ε","price":591.007} |
| PUT /items/324 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=324, body={"name":"ऒ𠀷","price":428.256,"quantity":627} |
| PUT /items/913 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=913, body={"name":"丁Х丳𠀑갾𠀰","price":472.905} |
| PUT /items/803 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=803, body={"description":"रÃभ𠁩","name":"Ν乐걁Й걥Ο갌걊𠀿𠁠","price":743.816,"quantity":139} |
| PUT /items/809 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=809, body={"name":"𠀈乎걆个و丠0갣ـ같","price":121.604,"quantity":739} |
| PUT /items/363 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=363, body={"name":"とぷ𠁝걘","price":576.176} |
| PUT /items/902 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=902, body={"description":"갬ぃШ","name":"갴á걔갧ج","price":661.385,"quantity":758} |
| PUT /items/971 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=971, body={"name":"2乲𠁄т丝𠁠갏为","price":284.483,"quantity":672} |
| PUT /items/688 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=688, body={"name":"𠁔𠀻X𠀵यÎ","price":969.216} |
| PUT /items/83 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=83, body={"name":"छд𠁋𠁺","price":854.462,"quantity":197} |
| PUT /items/327 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=327, body={"name":"乪且𠀪乙कA갆𠀣갃ф","price":947.548} |
| PUT /items/221 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=221, body={"name":"Ò个ぢ𠁞갌ؿõ","price":746.317} |
| PUT /items/501 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=501, body={"description":"之감てぺ","name":"书òゕ갬거d","price":579.4} |
| PUT /items/799 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=799, body={"name":"乥इγÀ丆","price":170.734} |
| PUT /items/720 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=720, body={"name":"эةがكdЩ乁걅","price":782.853,"quantity":584} |
| PUT /items/899 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=899, body={"name":"AふؿさÍ𠁿乫乥","price":879.046,"quantity":291} |
| PUT /items/483 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=483, body={"name":"Éぉ걳걟Ê𠀿𠁹ن","price":703.91} |
| PUT /items/668 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=668, body={"description":"乾𠁑","name":"𠁪で","price":944.802} |
| PUT /items/462 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=462, body={"description":"たД乵","name":"丰𠀃𠀿个Μ𠀡𠀪わί","price":363.632} |
| PUT /items/584 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=584, body={"description":"习ゅ","name":"串丙乐げ丽ゃþ","price":772.783,"quantity":153} |
| PUT /items/341 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=341, body={"description":"乣Р","name":"だеف","price":23.444,"quantity":996} |
| PUT /items/133 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=133, body={"name":"𠀫UبぃΫ乙w乲м걽","price":471.855,"quantity":417} |
| PUT /items/990 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=990, body={"name":"𠀉𠁤","price":351.557,"quantity":315} |
| PUT /items/986 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=986, body={"name":"थण丏","price":52.727} |
| PUT /items/109 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=109, body={"description":"Ý걠𠀔ñ가طaË값に","name":"𠁯う갟𠀉","price":6.013} |
| PUT /items/509 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=509, body={"description":"Λûنध걩갢九𠀐í걕","name":"έ갇УऒΥ","price":282.948,"quantity":925} |
| PUT /items/640 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=640, body={"name":"ぼ","price":75.617} |
| PUT /items/437 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=437, body={"name":"قФ乼갈ά걊𠁱π","price":321.029,"quantity":277} |
| PUT /items/246 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=246, body={"description":"걢Зο","name":"乏𠀵𠁎","price":17.049,"quantity":2} |
| PUT /items/909 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=909, body={"description":"で丑丱ç𠀦ど","name":"갍业갚","price":808.632} |
| PUT /items/364 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=364, body={"description":"ع𠁑","name":"æ걉ؿ걉îみ","price":436.149} |
| PUT /items/409 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=409, body={"name":"Eあش両𠀐づथ𠁍么","price":926.739,"quantity":447} |
| PUT /items/535 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=535, body={"name":"丬व","price":262.827} |
| PUT /items/943 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=943, body={"name":"झ且𠁰","price":110.615} |
| PUT /items/959 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=959, body={"description":"СЫ걉","name":"걦Πय걋𠀄わठï乨Ь","price":241.526} |
| PUT /items/182 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=182, body={"description":"丯и걮𠁙","name":"걭ÜごÂ𠁿ñЗ习","price":297.393,"quantity":573} |
| PUT /items/435 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=435, body={"description":"乱걋","name":"る","price":8.683} |
| PUT /items/333 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=333, body={"name":"Φつ丨О𠁕に𠁘Щご","price":479.546} |
| PUT /items/622 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=622, body={"description":"Ã","name":"걏下ػΝ乩걅걛ì习þ","price":276.986,"quantity":149} |
| PUT /items/904 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=904, body={"name":"Α","price":787.765} |
| PUT /items/238 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=238, body={"name":"𠁟","price":327.3,"quantity":382} |
| PUT /items/147 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=147, body={"name":"걮Ξ갃त객𠁓义𠀩","price":132.314} |
| PUT /items/43 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=43, body={"name":"갣0","price":561.092,"quantity":464} |
| PUT /items/236 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=236, body={"name":"ऌ东","price":726.147} |
| PUT /items/335 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=335, body={"description":"らथ","name":"ПÀ","price":905.592,"quantity":267} |
| PUT /items/789 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=789, body={"description":"さة갅が𠁴丣い갖𠀶","name":"걠щТ","price":926.316,"quantity":111} |
| PUT /items/637 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=637, body={"name":"らì丧갥Ь갡걛ЯÞ갆","price":862.404,"quantity":758} |
| PUT /items/438 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=438, body={"description":"乵Τß","name":"οt乔𠁋乏Uだ","price":453.408,"quantity":43} |
| PUT /items/319 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=319, body={"name":"m","price":581.59} |
| PUT /items/701 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=701, body={"description":"𠀩","name":"るΙ걡𠁶θ건Î丮へ","price":991.045,"quantity":164} |
| PUT /items/759 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=759, body={"name":"𠀘丅Υ𠀦乭걈乌","price":275.382,"quantity":888} |
| PUT /items/743 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=743, body={"name":"갷ऱÄख강b갛J","price":357.279,"quantity":847} |
| PUT /items/807 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=807, body={"description":"くが三ò𠁃ù","name":"𠀇ь𠀖𠁟主びفثか","price":551.524,"quantity":104} |
| PUT /items/36 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=36, body={"name":"m𠁇Ê갦","price":117.5,"quantity":444} |
| PUT /items/142 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=142, body={"name":"мøせκङ걹","price":921.656,"quantity":436} |
| PUT /items/343 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=343, body={"name":"걜द","price":652.321} |
| PUT /items/539 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=539, body={"description":"ऌ书𠀘e","name":"Β","price":796.773} |
| PUT /items/526 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=526, body={"name":"丞なÀ𠁹","price":641.107,"quantity":345} |
| PUT /items/253 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=253, body={"name":"久उ𠁻","price":121.263} |
| PUT /items/124 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=124, body={"description":"각𠀔ج𠁊걒𠀻𠁎ゐρμ","name":"ΚοК𠀗ち갅각𠀚갸Н","price":970.648,"quantity":614} |
| PUT /items/352 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=352, body={"name":"رचS걹걝Áゖए丹갺","price":170.382} |
| PUT /items/604 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=604, body={"description":"ά乊걓そ","name":"乓","price":286.246} |
| PUT /items/187 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=187, body={"name":"갞乐𠀩丫𠀢걺갏は갊","price":511.937,"quantity":907} |
| PUT /items/264 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=264, body={"name":"a𠀙べ丗û","price":540.165,"quantity":932} |
| PUT /items/961 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=961, body={"name":"А乨Üऒ乌nؽ丅걍","price":18.678,"quantity":5} |
| PUT /items/814 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=814, body={"description":"乯걝тþ","name":"걲𠀿Úю𠀙","price":809.569,"quantity":267} |
| PUT /items/716 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=716, body={"name":"ぢذ乷乔串걱j","price":919.923} |
| PUT /items/547 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=547, body={"description":"HろÄÅ𠁅ى","name":"걫𠀴걶","price":747.723} |

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

100 sent, 100 reported (filter: all). Status codes: ERR ×100.

| Endpoint | Status | Message | Input |
|---|---|---|---|
| DELETE /items/186 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=186 |
| DELETE /items/473 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=473 |
| DELETE /items/402 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=402 |
| DELETE /items/552 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=552 |
| DELETE /items/930 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=930 |
| DELETE /items/591 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=591 |
| DELETE /items/338 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=338 |
| DELETE /items/316 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=316 |
| DELETE /items/790 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=790 |
| DELETE /items/211 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=211 |
| DELETE /items/292 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=292 |
| DELETE /items/330 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=330 |
| DELETE /items/108 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=108 |
| DELETE /items/397 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=397 |
| DELETE /items/944 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=944 |
| DELETE /items/591 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=591 |
| DELETE /items/23 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=23 |
| DELETE /items/265 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=265 |
| DELETE /items/927 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=927 |
| DELETE /items/219 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=219 |
| DELETE /items/240 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=240 |
| DELETE /items/664 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=664 |
| DELETE /items/643 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=643 |
| DELETE /items/70 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=70 |
| DELETE /items/648 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=648 |
| DELETE /items/822 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=822 |
| DELETE /items/177 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=177 |
| DELETE /items/377 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=377 |
| DELETE /items/143 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=143 |
| DELETE /items/801 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=801 |
| DELETE /items/362 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=362 |
| DELETE /items/642 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=642 |
| DELETE /items/375 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=375 |
| DELETE /items/583 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=583 |
| DELETE /items/721 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=721 |
| DELETE /items/860 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=860 |
| DELETE /items/325 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=325 |
| DELETE /items/740 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=740 |
| DELETE /items/107 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=107 |
| DELETE /items/514 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=514 |
| DELETE /items/133 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=133 |
| DELETE /items/938 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=938 |
| DELETE /items/892 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=892 |
| DELETE /items/564 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=564 |
| DELETE /items/867 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=867 |
| DELETE /items/95 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=95 |
| DELETE /items/225 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=225 |
| DELETE /items/786 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=786 |
| DELETE /items/671 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=671 |
| DELETE /items/213 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=213 |
| DELETE /items/422 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=422 |
| DELETE /items/232 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=232 |
| DELETE /items/459 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=459 |
| DELETE /items/864 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=864 |
| DELETE /items/969 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=969 |
| DELETE /items/653 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=653 |
| DELETE /items/658 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=658 |
| DELETE /items/403 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=403 |
| DELETE /items/404 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=404 |
| DELETE /items/953 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=953 |
| DELETE /items/141 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=141 |
| DELETE /items/107 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=107 |
| DELETE /items/47 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=47 |
| DELETE /items/527 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=527 |
| DELETE /items/274 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=274 |
| DELETE /items/322 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=322 |
| DELETE /items/740 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=740 |
| DELETE /items/607 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=607 |
| DELETE /items/813 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=813 |
| DELETE /items/703 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=703 |
| DELETE /items/260 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=260 |
| DELETE /items/327 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=327 |
| DELETE /items/156 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=156 |
| DELETE /items/862 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=862 |
| DELETE /items/378 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=378 |
| DELETE /items/192 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=192 |
| DELETE /items/48 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=48 |
| DELETE /items/936 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=936 |
| DELETE /items/258 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=258 |
| DELETE /items/453 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=453 |
| DELETE /items/832 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=832 |
| DELETE /items/224 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=224 |
| DELETE /items/92 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=92 |
| DELETE /items/286 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=286 |
| DELETE /items/406 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=406 |
| DELETE /items/336 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=336 |
| DELETE /items/981 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=981 |
| DELETE /items/808 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=808 |
| DELETE /items/277 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=277 |
| DELETE /items/583 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=583 |
| DELETE /items/361 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=361 |
| DELETE /items/144 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=144 |
| DELETE /items/67 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=67 |
| DELETE /items/637 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=637 |
| DELETE /items/232 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=232 |
| DELETE /items/211 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=211 |
| DELETE /items/554 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=554 |
| DELETE /items/759 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=759 |
| DELETE /items/46 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=46 |
| DELETE /items/255 | ERR | dial tcp 127.0.0.1:8000: connect: connection refused | item\_id=255 |

