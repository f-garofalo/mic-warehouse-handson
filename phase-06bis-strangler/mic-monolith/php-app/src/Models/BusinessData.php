<?php
declare(strict_types=1);

namespace MIC\Models;

/**
 * Generic envelope for any record_type stored in business_data.
 * Voluntarily generic: typed accessors are responsibility of callers
 * (this is the very pain we will fix later by extracting BCs).
 */
final class BusinessData
{
    public function __construct(
        public ?int    $id          = null,
        public string  $recordType  = '',
        public ?string $code        = null,
        public ?string $name        = null,
        public ?string $description = null,
        public ?int    $parentId    = null,
        public ?string $parentType  = null,
        public ?float  $amount1     = null,
        public ?float  $amount2     = null,
        public ?float  $amount3     = null,
        public ?float  $amount4     = null,
        public ?string $date1       = null,
        public ?string $date2       = null,
        public ?string $date3       = null,
        public ?string $text1       = null,
        public ?string $text2       = null,
        public ?string $text3       = null,
        public ?string $text4       = null,
        public ?string $text5       = null,
        public ?string $status      = null,
        public ?array  $payload     = null,
        public ?string $createdAt   = null,
        public ?string $updatedAt   = null,
    ) {}

    public static function fromRow(array $r): self
    {
        return new self(
            id:          isset($r['id']) ? (int)$r['id'] : null,
            recordType:  (string)($r['record_type'] ?? ''),
            code:        $r['code']        ?? null,
            name:        $r['name']        ?? null,
            description: $r['description'] ?? null,
            parentId:    isset($r['parent_id']) ? (int)$r['parent_id'] : null,
            parentType:  $r['parent_type']  ?? null,
            amount1:     isset($r['amount_1']) ? (float)$r['amount_1'] : null,
            amount2:     isset($r['amount_2']) ? (float)$r['amount_2'] : null,
            amount3:     isset($r['amount_3']) ? (float)$r['amount_3'] : null,
            amount4:     isset($r['amount_4']) ? (float)$r['amount_4'] : null,
            date1:       $r['date_1'] ?? null,
            date2:       $r['date_2'] ?? null,
            date3:       $r['date_3'] ?? null,
            text1:       $r['text_1'] ?? null,
            text2:       $r['text_2'] ?? null,
            text3:       $r['text_3'] ?? null,
            text4:       $r['text_4'] ?? null,
            text5:       $r['text_5'] ?? null,
            status:      $r['status'] ?? null,
            payload:     isset($r['payload_json']) && $r['payload_json']
                            ? (json_decode((string)$r['payload_json'], true) ?: null)
                            : null,
            createdAt:   $r['created_at'] ?? null,
            updatedAt:   $r['updated_at'] ?? null,
        );
    }

    public function toArray(): array
    {
        return [
            'id'           => $this->id,
            'record_type'  => $this->recordType,
            'code'         => $this->code,
            'name'         => $this->name,
            'description'  => $this->description,
            'parent_id'    => $this->parentId,
            'parent_type'  => $this->parentType,
            'amount_1'     => $this->amount1,
            'amount_2'     => $this->amount2,
            'amount_3'     => $this->amount3,
            'amount_4'     => $this->amount4,
            'date_1'       => $this->date1,
            'date_2'       => $this->date2,
            'date_3'       => $this->date3,
            'text_1'       => $this->text1,
            'text_2'       => $this->text2,
            'text_3'       => $this->text3,
            'text_4'       => $this->text4,
            'text_5'       => $this->text5,
            'status'       => $this->status,
            'payload'      => $this->payload,
            'created_at'   => $this->createdAt,
            'updated_at'   => $this->updatedAt,
        ];
    }
}
