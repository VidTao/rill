export function load({ params, url }: { params: { platform: string }; url: URL }) {
  const searchParams = Object.fromEntries(url.searchParams.entries());
  return {
    platform: params.platform,
    params: searchParams,
  };
}
