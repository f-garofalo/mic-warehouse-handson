<?php
declare(strict_types=1);

namespace MIC\Controllers;

final class AuditController extends BaseController
{
    protected function type(): string { return 'audit_log'; }

    protected function toDto(array $r): array
    {
        return [
            'id'          => $r['id'],
            'codice'      => $r['code'],
            'descrizione' => $r['name'],
            'utente_id'   => $r['parent_id'],
            'data'        => $r['date_1'],
            'azione'      => $r['text_1'],
            'target_type' => $r['text_2'],
            'ip'          => $r['payload']['ip'] ?? null,
            'status'      => $r['status'],
        ];
    }
}
