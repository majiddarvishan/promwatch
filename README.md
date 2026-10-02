# promwatch

## without label

```bash
./promwatch \
  --url http://localhost:8080/metrics \
  --metric sms_messages_total \
  --rate \
  --interval 1s
```

## with one label

```bash
./promwatch \
  --url http://localhost:8080/metrics \
  --metric 'sms_messages_total{direction="mt"}' \
  --rate \
  --interval 1s
```

## with multiple label

```bash
./promwatch \
  --url http://localhost:8080/metrics \
  --metric 'sms_messages_total{direction="mt",operator="mci"}' \
  --rate \
  --interval 1s
```
