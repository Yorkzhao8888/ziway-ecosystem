// 由 oapi-codegen 从 00_OpenAPI.yaml 生成的 Client（精简版，与 Go api/types.go 1:1）
export type RootRole = 'OU' | 'HU' | 'EU' | 'GU';

export interface ErrorResponse {
  code: string;
  message: string;
  details?: unknown;
  traceId?: string;
}

export interface Resource {
  id: string;
  tenantId: string;
  kind: string; // eos/ems/cos/...
  data?: unknown;
}

export class ZiwayClient {
  constructor(
    private baseURL: string = 'http://localhost:8080',
    private tenantId: string = 'demo',
  ) {}

  /** 通用资源调用（GET/POST /api/{kind}/{tenantId}/resources） */
  async resource(kind: string, method: 'GET' | 'POST' = 'GET', body?: unknown): Promise<Resource | { [k: string]: unknown }> {
    const res = await fetch(`${this.baseURL}/api/${kind}/${this.tenantId}/resources`, {
      method,
      headers: { 'Content-Type': 'application/json', 'X-Tenant-Id': this.tenantId },
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!res.ok) {
      const err = (await res.json()) as ErrorResponse;
      throw new Error(`[${err.code}] ${err.message}`);
    }
    return res.json();
  }

  // ---- SC-001 便捷方法 ----
  registerEnterprise(name: string, rootRole: RootRole, licenseNo: string) {
    return this.resource('hos', 'POST', { name, root_role: rootRole, license_no: licenseNo });
  }
  applyMarket(category: string, capacity: number, deliveryArea: string) {
    return this.resource('eos', 'POST', { category, capacity, delivery_area: deliveryArea });
  }
  createProduct(shopId: number, sku: string, name: string, price: number, stock: number) {
    return this.resource('eos', 'POST', { shop_id: shopId, sku, name, price, stock });
  }
  createOrder(supplierTenantId: string, items: { product_id: number; qty: number }[]) {
    return this.resource('cos', 'POST', { supplier_tenant_id: supplierTenantId, items });
  }
  submitReview(orderId: number, shopId: number, rating: number, comment: string) {
    return fetch(`${this.baseURL}/api/cos/${this.tenantId}/reviews`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Tenant-Id': this.tenantId },
      body: JSON.stringify({ order_id: orderId, shop_id: shopId, rating, comment }),
    }).then((r) => r.json());
  }
  withdraw(settlementId: number) {
    return this.resource('fos', 'POST', { settlement_id: settlementId });
  }
  metrics() {
    return this.resource('vos', 'GET');
  }
}
