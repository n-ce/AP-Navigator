export async function GET(request) {
  const api = 'https://acharyaprashant.org/api/v2/content/';
  const path = 'index?contentType=6&lf=2&limit=400&offset=0';

  const res = await fetch(api + path)
    .then(res => res.json())
    .then(data => data.contents.data.map(book => ({
      id: book.id,
      title: book.title?.english,
      description: book.description?.english
    })));


  return new Response(
    JSON.stringify(res), {
    headers: { 'Content-Type': 'application/json' },
  });
}
