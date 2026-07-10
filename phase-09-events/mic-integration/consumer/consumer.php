<?php
declare(strict_types=1);

$mode = strtolower(trim(getenv('CONSUMER_MODE') ?: 'once'));
$interval = max(1, (int)(getenv('POLL_INTERVAL_SECONDS') ?: '5'));
$recordsUrl = getenv('HERMES_RECORDS_URL') ?: 'http://warehouse-bc:8081/debug/hermes/records?type=warehouse.article.v1';

$pdo = new PDO(
    sprintf(
        'mysql:host=%s;port=%s;dbname=%s;charset=utf8mb4',
        getenv('DB_HOST') ?: 'integration-mysql',
        getenv('DB_PORT') ?: '3306',
        getenv('DB_NAME') ?: 'mic'
    ),
    getenv('DB_USER') ?: 'root',
    getenv('DB_PASS') ?: 'root',
    [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION, PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC]
);

if ($mode === 'loop') {
    echo "MIC Hermes consumer started in loop mode (interval={$interval}s, url={$recordsUrl})\n";
    while (true) {
        try {
            $projected = projectOnce($pdo, $recordsUrl);
            echo sprintf("[%s] projected %d warehouse article record(s) into MIC read model\n", date(DATE_ATOM), $projected);
        } catch (Throwable $e) {
            fwrite(STDERR, sprintf("[%s] consumer error: %s\n", date(DATE_ATOM), $e->getMessage()));
        }
        sleep($interval);
    }
}

$projected = projectOnce($pdo, $recordsUrl);
echo "OK: projected {$projected} warehouse article record(s) into MIC read model\n";

function projectOnce(PDO $pdo, string $recordsUrl): int
{
    $json = @file_get_contents($recordsUrl);
    if ($json === false) {
        throw new RuntimeException("Unable to read Hermes mock records from {$recordsUrl}");
    }
    $records = json_decode($json, true);
    if (!is_array($records)) {
        throw new RuntimeException('Hermes mock returned invalid JSON');
    }

    $projected = 0;
    foreach ($records as $record) {
        if (($record['type'] ?? '') !== 'warehouse.article.v1') {
            continue;
        }
        $data = $record['data'] ?? [];
        $sku = trim((string)($data['sku'] ?? ''));
        if ($sku === '') {
            continue;
        }

        $projection = mapHermesArticleToMicProjection($record);
        upsertProjection($pdo, $projection);
        $projected++;
    }

    return $projected;
}

function mapHermesArticleToMicProjection(array $record): array
{
    // TODO Phase 09 Task 2 — complete only this function: the consumer-side
    // adapter between the Data Product and the MIC read model.
    //
    // Input: one CloudEvents record: envelope fields at the top level (id,
    // type, source, subject, time) and the Data Product payload in
    // $record['data'] (article_id, sku, name, price_cents, currency).
    //
    // Output: the array upsertProjection() persists, with keys:
    // - code = data.sku
    // - name = data.name, falling back to data.sku when empty
    // - description = data.description, falling back to ''
    // - amount_1 = price_cents as a decimal string: centsToDecimal((int)...),
    //   never a float
    // - amount_2 = 1, text_1 = 'WAREHOUSE-BC', text_2 = 'IVA22',
    //   status = 'attivo' (the defaults MIC expects for this projection)
    // - payload_json = JSON preserving projection_kind ('warehouse_article'),
    //   article_id, currency, and the envelope metadata as source_record_id,
    //   source_record_type, source, source_subject, source_time: the audit
    //   trail that links every projected row back to its Hermes record.
    throw new RuntimeException('TODO Phase 09 Task 2: implement Hermes article -> MIC projection mapping');
}

function upsertProjection(PDO $pdo, array $projection): void
{
    $existing = findProjection($pdo, (string)$projection['code']);
    if ($existing) {
        $stmt = $pdo->prepare(
            'UPDATE business_data
                SET name = ?, description = ?, amount_1 = ?, amount_2 = ?, text_1 = ?, text_2 = ?, status = ?, payload_json = ?, updated_at = CURRENT_TIMESTAMP
              WHERE id = ?'
        );
        $stmt->execute([
            $projection['name'],
            $projection['description'],
            $projection['amount_1'],
            $projection['amount_2'],
            $projection['text_1'],
            $projection['text_2'],
            $projection['status'],
            $projection['payload_json'],
            (int)$existing['id'],
        ]);
        return;
    }

    $stmt = $pdo->prepare(
        'INSERT INTO business_data (record_type, code, name, description, amount_1, amount_2, text_1, text_2, status, payload_json)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)'
    );
    $stmt->execute([
        'warehouse_article_projection',
        $projection['code'],
        $projection['name'],
        $projection['description'],
        $projection['amount_1'],
        $projection['amount_2'],
        $projection['text_1'],
        $projection['text_2'],
        $projection['status'],
        $projection['payload_json'],
    ]);
}

function findProjection(PDO $pdo, string $sku): ?array
{
    $stmt = $pdo->prepare('SELECT * FROM business_data WHERE record_type = ? AND code = ? LIMIT 1');
    $stmt->execute(['warehouse_article_projection', $sku]);
    $row = $stmt->fetch();
    return $row ?: null;
}

function centsToDecimal(int $cents): string
{
    $sign = $cents < 0 ? '-' : '';
    $abs = abs($cents);
    return sprintf('%s%d.%02d', $sign, intdiv($abs, 100), $abs % 100);
}
