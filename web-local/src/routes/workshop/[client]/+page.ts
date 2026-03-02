export function load({ params }: { params: { client: string } }) {
  return { clientName: params.client };
}
