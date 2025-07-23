export async function GET(request) {
  const api = 'https://acharyaprashant.org/api/v2/content/';
  const id = new URL(request.url).searchParams.get('id');
  const res = await fetch(api + 'search', {
    method: 'POST',
    headers: {
      'X-Client-Type': 'web',
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      q: id,
      sft: false,
      limitTypes: [1],
      offset: '',
      lf: 2,
      limit: 1,
      forceSearchTerm: false
    })
  })
    .then(res => res.json())
    .then(data => {
      const host = 'https://acharyaprashant.org/en/articles/';
      const seoSlug = data.searchedContents.data[0]?.meta?.seoSlug;
      return host + seoSlug;
    });

  return new Response(res);
}
