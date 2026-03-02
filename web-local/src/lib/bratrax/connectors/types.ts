export type ConnectionCategory =
  | "advertising"
  | "ecommerce"
  | "crm"
  | "analytics";

export type FlowType =
  | "oauth-redirect"
  | "oauth-redirect-region"
  | "oauth-modal-input"
  | "credential-modal"
  | "client-sdk";

export interface PlatformConfig {
  id: string;
  name: string;
  logo: string;
  description: string;
  category: ConnectionCategory;
  flowType: FlowType;
  apiSlug: string;
  callbackParams: string[];
  connectionKey: string;
  detailType: "ad" | "crm" | "none";
  regions?: string[];
}

export interface PlatformConnection {
  created_at: string;
  updated_at: string;
  platform: string;
}

export interface AdvertisingConnection {
  accountName: string;
  accountId: string;
  currency: string;
  timezone: string;
  createdAt: string;
}

export interface CrmConnection {
  storeUrl: string;
  storeName: string;
  currency: string;
  timezone: string;
  createdAt: string;
}
