<?php
declare(strict_types=1);

namespace MIC\Models;

final class BusinessRelation
{
    public function __construct(
        public ?int    $id           = null,
        public int     $sourceId     = 0,
        public int     $targetId     = 0,
        public string  $relationType = '',
        public ?float  $amount       = null,
        public ?array  $metadata     = null,
        public ?string $createdAt    = null,
    ) {}

    public static function fromRow(array $r): self
    {
        return new self(
            id:           isset($r['id']) ? (int)$r['id'] : null,
            sourceId:     (int)($r['source_id'] ?? 0),
            targetId:     (int)($r['target_id'] ?? 0),
            relationType: (string)($r['relation_type'] ?? ''),
            amount:       isset($r['amount']) ? (float)$r['amount'] : null,
            metadata:     isset($r['metadata']) && $r['metadata']
                              ? (json_decode((string)$r['metadata'], true) ?: null)
                              : null,
            createdAt:    $r['created_at'] ?? null,
        );
    }

    public function toArray(): array
    {
        return [
            'id'            => $this->id,
            'source_id'     => $this->sourceId,
            'target_id'     => $this->targetId,
            'relation_type' => $this->relationType,
            'amount'        => $this->amount,
            'metadata'      => $this->metadata,
            'created_at'    => $this->createdAt,
        ];
    }
}
