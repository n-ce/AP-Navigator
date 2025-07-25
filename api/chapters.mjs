export async function GET(request) {
  const api = 'https://acharyaprashant.org/api/v2/content/';
  const id = new URL(request.url).searchParams.get('id');
  const res = await fetch(
    api + id + '?lf=0', {
    method: 'GET',
    headers: {
      'X-Client-Type': 'web'
    }
  })
    .then(res => res.json())
    .then(data => data.content.enumMask.subContents["1"].value.chapters.map(chapter => chapter.title));

  return new Response(
    JSON.stringify(res), {
    headers: { 'Content-Type': 'application/json' },
  });
}
