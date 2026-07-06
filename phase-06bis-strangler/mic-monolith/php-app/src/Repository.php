<?php
declare(strict_types=1);

namespace MIC;

use PDO;
use MIC\Models\BusinessData;
use MIC\Models\BusinessRelation;

/**
 * Generic repository over business_data + business_relations.
 *
 * The whole monolith routes through here: the controllers know what
 * each amount_N / text_N "really means" for each record_type. That is
 * the anti-pattern we want to make visible.
 */
final class Repository
{
    private PDO $pdo;

    /** Whitelist of columns clients can filter on (avoids SQL injection in keys). */
    private const FILTER_COLS = [
        'id','code','name','description','parent_id','parent_type',
        'amount_1','amount_2','amount_3','amount_4',
        'date_1','date_2','date_3',
        'text_1','text_2','text_3','text_4','text_5',
        'status',
    ];

    public function __construct(?PDO $pdo = null)
    {
        $this->pdo = $pdo ?? Database::get();
    }

    // --------------------------------------------------------------
    // QUERY
    // --------------------------------------------------------------

    /**
     * Find records by type with optional filters and pagination.
     *
     * @param array<string,mixed> $filters  exact-match filters on FILTER_COLS
     * @param array{limit?:int,offset?:int,search?:string,order?:string,dir?:string} $opts
     * @return array{data: BusinessData[], total: int}
     */
    public function findByType(string $type, array $filters = [], array $opts = []): array
    {
        $where  = ['record_type = :type'];
        $params = [':type' => $type];

        foreach ($filters as $k => $v) {
            if (!in_array($k, self::FILTER_COLS, true)) continue;
            if ($v === null || $v === '') continue;
            $ph = ':f_' . $k;
            $where[] = "$k = $ph";
            $params[$ph] = $v;
        }

        if (!empty($opts['search'])) {
            $where[] = "(code LIKE :s_search OR name LIKE :s_search OR description LIKE :s_search OR text_1 LIKE :s_search)";
            $params[':s_search'] = '%' . $opts['search'] . '%';
        }

        $whereSql = 'WHERE ' . implode(' AND ', $where);

        // Total count
        $stmt = $this->pdo->prepare("SELECT COUNT(*) c FROM business_data $whereSql");
        $stmt->execute($params);
        $total = (int) $stmt->fetchColumn();

        // Order
        $orderCol = in_array($opts['order'] ?? '', self::FILTER_COLS, true) ? $opts['order'] : 'id';
        $dir = (strtoupper($opts['dir'] ?? 'DESC') === 'ASC') ? 'ASC' : 'DESC';

        $limit  = max(1, min(500, (int)($opts['limit']  ?? 50)));
        $offset = max(0, (int)($opts['offset'] ?? 0));

        $sql = "SELECT * FROM business_data $whereSql ORDER BY $orderCol $dir LIMIT $limit OFFSET $offset";
        $stmt = $this->pdo->prepare($sql);
        $stmt->execute($params);
        $rows = $stmt->fetchAll();

        return [
            'data'  => array_map([BusinessData::class, 'fromRow'], $rows),
            'total' => $total,
        ];
    }

    public function findById(int $id): ?BusinessData
    {
        $stmt = $this->pdo->prepare("SELECT * FROM business_data WHERE id = ?");
        $stmt->execute([$id]);
        $row = $stmt->fetch();
        return $row ? BusinessData::fromRow($row) : null;
    }

    // --------------------------------------------------------------
    // WRITE
    // --------------------------------------------------------------

    /**
     * Insert a new record_type record. $data is column => value;
     * unknown keys are silently ignored.
     */
    public function create(string $type, array $data): BusinessData
    {
        $payload = $data['payload'] ?? null;
        unset($data['payload']);

        $cols = ['record_type'];
        $vals = [':record_type'];
        $bind = [':record_type' => $type];

        foreach (self::FILTER_COLS as $col) {
            if ($col === 'id') continue;
            if (array_key_exists($col, $data)) {
                $cols[] = $col;
                $vals[] = ':' . $col;
                $bind[':' . $col] = $data[$col] === '' ? null : $data[$col];
            }
        }

        if ($payload !== null) {
            $cols[] = 'payload_json';
            $vals[] = ':payload_json';
            $bind[':payload_json'] = is_string($payload) ? $payload : json_encode($payload);
        }

        $sql = 'INSERT INTO business_data (' . implode(',', $cols) . ') VALUES (' . implode(',', $vals) . ')';
        $stmt = $this->pdo->prepare($sql);
        $stmt->execute($bind);
        $id = (int)$this->pdo->lastInsertId();

        return $this->findById($id) ?? throw new \RuntimeException('Insert failed');
    }

    public function update(int $id, array $data): ?BusinessData
    {
        $payload = $data['payload'] ?? null;
        unset($data['payload']);

        $set = [];
        $bind = [':id' => $id];

        foreach (self::FILTER_COLS as $col) {
            if ($col === 'id') continue;
            if (array_key_exists($col, $data)) {
                $set[] = "$col = :$col";
                $bind[':' . $col] = $data[$col] === '' ? null : $data[$col];
            }
        }
        if ($payload !== null) {
            $set[] = 'payload_json = :payload_json';
            $bind[':payload_json'] = is_string($payload) ? $payload : json_encode($payload);
        }

        if (!$set) return $this->findById($id);

        $sql = 'UPDATE business_data SET ' . implode(',', $set) . ' WHERE id = :id';
        $stmt = $this->pdo->prepare($sql);
        $stmt->execute($bind);
        return $this->findById($id);
    }

    public function delete(int $id): bool
    {
        $this->pdo->prepare('DELETE FROM business_relations WHERE source_id = ? OR target_id = ?')
                  ->execute([$id, $id]);
        $stmt = $this->pdo->prepare('DELETE FROM business_data WHERE id = ?');
        $stmt->execute([$id]);
        return $stmt->rowCount() > 0;
    }

    // --------------------------------------------------------------
    // RELATIONS
    // --------------------------------------------------------------

    public function relate(int $sourceId, int $targetId, string $type, ?float $amount = null, ?array $metadata = null): BusinessRelation
    {
        $stmt = $this->pdo->prepare(
            'INSERT INTO business_relations (source_id, target_id, relation_type, amount, metadata)
             VALUES (?, ?, ?, ?, ?)'
        );
        $stmt->execute([
            $sourceId, $targetId, $type, $amount,
            $metadata !== null ? json_encode($metadata) : null,
        ]);
        $id = (int)$this->pdo->lastInsertId();
        return new BusinessRelation(
            id: $id, sourceId: $sourceId, targetId: $targetId,
            relationType: $type, amount: $amount, metadata: $metadata,
        );
    }

    /** First target_id for given source/relation_type. Useful for 1-1 rels. */
    public function targetIdOf(int $sourceId, string $type): ?int
    {
        $stmt = $this->pdo->prepare(
            'SELECT target_id FROM business_relations
              WHERE source_id = ? AND relation_type = ?
              ORDER BY id DESC LIMIT 1'
        );
        $stmt->execute([$sourceId, $type]);
        $r = $stmt->fetchColumn();
        return $r === false ? null : (int)$r;
    }

    public function sourceIdOf(int $targetId, string $type): ?int
    {
        $stmt = $this->pdo->prepare(
            'SELECT source_id FROM business_relations
              WHERE target_id = ? AND relation_type = ?
              ORDER BY id DESC LIMIT 1'
        );
        $stmt->execute([$targetId, $type]);
        $r = $stmt->fetchColumn();
        return $r === false ? null : (int)$r;
    }

    /** All relations originating from $sourceId (optionally filtered by type). */
    public function relationsFrom(int $sourceId, ?string $type = null): array
    {
        if ($type) {
            $stmt = $this->pdo->prepare(
                'SELECT * FROM business_relations WHERE source_id = ? AND relation_type = ? ORDER BY id'
            );
            $stmt->execute([$sourceId, $type]);
        } else {
            $stmt = $this->pdo->prepare(
                'SELECT * FROM business_relations WHERE source_id = ? ORDER BY id'
            );
            $stmt->execute([$sourceId]);
        }
        return array_map([BusinessRelation::class,'fromRow'], $stmt->fetchAll());
    }

    public function relationsTo(int $targetId, ?string $type = null): array
    {
        if ($type) {
            $stmt = $this->pdo->prepare(
                'SELECT * FROM business_relations WHERE target_id = ? AND relation_type = ? ORDER BY id'
            );
            $stmt->execute([$targetId, $type]);
        } else {
            $stmt = $this->pdo->prepare(
                'SELECT * FROM business_relations WHERE target_id = ? ORDER BY id'
            );
            $stmt->execute([$targetId]);
        }
        return array_map([BusinessRelation::class,'fromRow'], $stmt->fetchAll());
    }

    public function deleteRelation(int $relId): bool
    {
        $stmt = $this->pdo->prepare('DELETE FROM business_relations WHERE id = ?');
        $stmt->execute([$relId]);
        return $stmt->rowCount() > 0;
    }

    // --------------------------------------------------------------
    // CONVENIENCE / RAW
    // --------------------------------------------------------------

    public function pdo(): PDO { return $this->pdo; }

    public function rawAll(string $sql, array $params = []): array
    {
        $stmt = $this->pdo->prepare($sql);
        $stmt->execute($params);
        return $stmt->fetchAll();
    }

    public function rawOne(string $sql, array $params = []): ?array
    {
        $stmt = $this->pdo->prepare($sql);
        $stmt->execute($params);
        $r = $stmt->fetch();
        return $r ?: null;
    }
}
