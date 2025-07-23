

const fetchFromApApi = async (endpoint, options = {}) => {
  const apApiBase = 'https://acharyaprashant.org/api/v2/content/';
  const url = `${apApiBase}${endpoint}`;
  const finalOptions = {
    ...options,
    headers: {
      'X-Client-Type': 'web',
      ...options.headers,
    },
  };

  const response = await fetch(url, finalOptions);

  if (!response.ok) {
    let errorDetails = `API responded with status ${response.status}`;
    try {
      const errorBody = await response.json();
      errorDetails += `: ${JSON.stringify(errorBody)}`;
    } catch (parseError) {
      errorDetails += `: ${response.statusText}`;
    }
    throw new Error(errorDetails);
  }

  return response.json();
};

export function GET(req, res) {
  const requestPath = req.url.split('?')[0];
  const { id } = req.query;

  try {
    if (req.method !== 'GET') { res.status(405).json({ error: 'Method Not Allowed' }); return; }

    if (requestPath === '/article') {

      if (!id) {
        res.status(400)
          .json({ error: 'Query parameter "id" (search query) is required for article search.' });
        return;
      }

      const searchResult = await fetchFromApApi('search', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          q: id,
          sft: false,
          limitTypes: [1],
          offset: '',
          lf: 2,
          limit: 5,
          forceSearchTerm: false
        }),
      });

      const host = 'https://acharyaprashant.org/en/articles/';
      const seoSlug = searchResult.searchedContents.data[0]?.meta?.seoSlug;
      const articleUrl = host + seoSlug;

      if (seoSlug)
        res.status(200)
          .json({ articleUrl });
      else
        res.status(404)
          .json({ message: 'No article found for the given query.' });


    } else if (requestPath === '/books') {

      const booksData = await fetchFromApApi('index?contentType=6&lf=2&limit=400&offset=0', { method: 'GET' });
      const refinedBooks = booksData.contents.data.map(book => ({
        id: book.id,
        title: book.title?.english,
        description: book.description?.english,
      }));
      res.status(200)
        .json(refinedBooks);

    } else if (requestPath === '/chapters') {

      if (!id) {
        res.status(400).json({ error: 'Query parameter "id" (book ID) is required for chapters.' });
        return;
      }
      const chaptersData = await fetchFromApApi(`${id}?lf=0`, { method: 'GET' });
      const refinedChapters = chaptersData.content.enumMask.subContents["1"].value.chapters.map(chapter => ({
        title: chapter.title,
      }));
      res.status(200)
        .json(refinedChapters);

    } else
      res.status(404)
        .json({ error: 'Not Found' });

  } catch (error) {
    console.error('API Handler Error:', error);
    res.status(500)
      .json({ error: 'An unexpected error occurred', details: error.message });
  }
};

