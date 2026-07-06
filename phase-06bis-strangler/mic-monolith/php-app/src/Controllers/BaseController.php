<?php
declare(strict_types=1);

namespace MIC\Controllers;

use MIC\Repository;

/**
 * Shared CRUD plumbing: the controllers below extend this and lean on
 * `findByType / create / update / delete` for the boring 80%.
 * The interesting business stuff lives in each concrete controller.
 */
abstract class BaseController
{
    protected Repository $repo;

    public function __construct(?Repository $repo = null)
    {
        $this->repo = $repo ?? new Repository();
    }

    /** record_type slug stored in business_data. */
    abstract protected function type(): string;

    /**
     * Map a business_data row to a friendly DTO for the entity.
     * Override in concrete controllers to label the generic columns.
     */
    abstract protected function toDto(array $row): array;

    /**
     * Map an incoming API payload to business_data columns.
     * Override per entity to translate friendly names back into amount_N/text_N.
     * Defaults to identity (callers must already use raw column names).
     */
    protected function fromPayload(array $body): array
    {
        return $body;
    }

    // ---------- generic JSON CRUD ----------

    public function index(): array
    {
        $q     = $_GET['q']      ?? null;
        $limit = (int)($_GET['limit']  ?? 50);
        $offset= (int)($_GET['offset'] ?? 0);
        $status= $_GET['status'] ?? null;
        $order = $_GET['order']  ?? 'id';
        $dir   = $_GET['dir']    ?? 'DESC';

        $filters = [];
        if ($status) $filters['status'] = $status;

        $res = $this->repo->findByType($this->type(), $filters, [
            'limit'  => $limit,
            'offset' => $offset,
            'search' => $q,
            'order'  => $order,
            'dir'    => $dir,
        ]);

        return [
            'data' => array_map(
                fn($bd) => $this->toDto($bd->toArray()),
                $res['data']
            ),
            'meta' => ['total' => $res['total'], 'limit' => $limit, 'offset' => $offset],
        ];
    }

    public function show(int $id): array
    {
        $b = $this->repo->findById($id);
        if (!$b || $b->recordType !== $this->type()) {
            return [404, ['error' => 'not_found']];
        }
        return ['data' => $this->toDto($b->toArray())];
    }

    public function create(): array
    {
        $body = $this->readJsonBody();
        if (!$body) return [400, ['error' => 'invalid_body']];
        $cols = $this->fromPayload($body);
        $b = $this->repo->create($this->type(), $cols);
        return [201, ['data' => $this->toDto($b->toArray())]];
    }

    public function update(int $id): array
    {
        $b = $this->repo->findById($id);
        if (!$b || $b->recordType !== $this->type()) {
            return [404, ['error' => 'not_found']];
        }
        $body = $this->readJsonBody();
        if (!$body) return [400, ['error' => 'invalid_body']];
        $cols = $this->fromPayload($body);
        $u = $this->repo->update($id, $cols);
        return ['data' => $this->toDto($u->toArray())];
    }

    public function destroy(int $id): array
    {
        $b = $this->repo->findById($id);
        if (!$b || $b->recordType !== $this->type()) {
            return [404, ['error' => 'not_found']];
        }
        $this->repo->delete($id);
        return [200, ['data' => ['id' => $id, 'deleted' => true]]];
    }

    protected function readJsonBody(): ?array
    {
        $raw = file_get_contents('php://input') ?: '';
        if ($raw === '') return null;
        $j = json_decode($raw, true);
        return is_array($j) ? $j : null;
    }
}
