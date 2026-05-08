export const GOOGLE_TRACKING_TEMPLATE = "{lpurl}";

export const GOOGLE_FINAL_URL_SUFFIX =
  "utm_source=google&utm_medium=cpc&utm_campaign={campaignid}&utm_term={adgroupid}&utm_content={creative}&bt_campaign_id={campaignid}&bt_asset_group_id={assetgroupid}&bt_ad_group_id={adgroupid}&bt_ad_id={creative}&bt_network={network}&bt_device={device}&bt_product_id={product_id}&bt_product_partition_id={product_partition_id}&campaign_id={campaignid}&ad_group_id={adgroupid}&asset_group_id={assetgroupid}&ad_id={creative}&network={network}&device={device}&matchtype={matchtype}&keyword={keyword}&placement={placement}&targetid={targetid}&adtype={adtype}&merchant_id={merchant_id}&product_id={product_id}&product_partition_id={product_partition_id}";

export const GOOGLE_ACCOUNT_TRACKING_TEMPLATE = `${GOOGLE_TRACKING_TEMPLATE}?${GOOGLE_FINAL_URL_SUFFIX}`;

export const META_URL_PARAMETERS =
  "utm_source=facebook&utm_medium=paid_social&utm_campaign={{campaign.id}}&utm_term={{adset.id}}&utm_content={{ad.id}}&bt_campaign_id={{campaign.id}}&bt_adset_id={{adset.id}}&bt_ad_id={{ad.id}}&bt_placement={{placement}}&bt_site_source_name={{site_source_name}}&campaign_id={{campaign.id}}&campaign_name={{campaign.name}}&adset_id={{adset.id}}&adset_name={{adset.name}}&ad_id={{ad.id}}&ad_name={{ad.name}}&site_source_name={{site_source_name}}&placement={{placement}}";

export const trackingVerificationChecklist = [
  "Click a test ad or preview link.",
  "Confirm the URL contains campaign, ad set or ad group, and ad IDs.",
  "Confirm the Bratrax dashboard resolves the campaign.",
  "Confirm lower-level ad set, asset group, or ad appears where supported.",
];

export const googleTrackingWarnings = [
  "Keep Google auto-tagging enabled.",
  "Paste this into the Tracking template field in Google Ads account global URL options.",
  "This combined value includes the landing page token and the full parameter string.",
];

export const metaTrackingWarnings = [
  "Do not include a leading ?.",
  "Paste this into URL Parameters, not Website URL.",
  "Apply this at campaign level in Meta Ads Manager; Meta does not reliably support this as an account-level URL setting.",
];
