package loader

import (
	"context"
	"sync"

	"github.com/nicksrandall/dataloader"

	"github.com/kevinclcn/graphql-go-example/errors"
	"github.com/kevinclcn/graphql-go-example/swapi"
)

func LoadFilm(ctx context.Context, url string) (swapi.Film, error) {
	var film swapi.Film

	ldr, err := extract(ctx, filmLoaderKey)
	if err != nil {
		return film, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(url))()
	if err != nil {
		return film, err
	}

	film, ok := data.(swapi.Film)
	if !ok {
		return film, errors.WrongType(film, data)
	}

	return film, nil
}

func LoadFilms(ctx context.Context, urls []string) (FilmResults, error) {
	var results []FilmResult
	ldr, err := extract(ctx, filmLoaderKey)
	if err != nil {
		return results, err
	}

	data, errs := ldr.LoadMany(ctx, dataloader.NewKeysFromStrings(urls))()
	results = make([]FilmResult, 0, len(urls))

	for i, d := range data {
		var e error
		if errs != nil {
			e = errs[i]
		}

		film, ok := d.(swapi.Film)
		if !ok && e == nil {
			e = errors.WrongType(film, d)
		}

		results = append(results, FilmResult{Film: film, Error: e})
	}

	return results, nil
}

// FilmResult is the (data, error) pair result of loading a specific key.
type FilmResult struct {
	Film  swapi.Film
	Error error
}

// FilmResults is a named type, so methods can be attached to []FilmResult.
type FilmResults []FilmResult

// WithoutErrors filters any result pairs with non-nil errors.
func (results FilmResults) WithoutErrors() []swapi.Film {
	var films = make([]swapi.Film, 0, len(results))

	for _, r := range results {
		if r.Error != nil {
			continue
		}

		films = append(films, r.Film)
	}

	return films
}

func PrimeFilms(ctx context.Context, page swapi.FilmPage) error {
	ldr, err := extract(ctx, filmLoaderKey)
	if err != nil {
		return err
	}

	for _, f := range page.Films {
		ldr.Prime(ctx, dataloader.StringKey(f.URL), f)
	}

	return nil
}

type filmGetter interface {
	Film(ctx context.Context, url string) (swapi.Film, error)
}

// FilmLoader contains the client required to load film resources.
type filmLoader struct {
	get filmGetter
}

func newFilmLoader(client filmGetter) dataloader.BatchFunc {
	return filmLoader{get: client}.loadBatch
}

func (ldr filmLoader) loadBatch(ctx context.Context, urls dataloader.Keys) []*dataloader.Result {
	var (
		n       = len(urls)
		results = make([]*dataloader.Result, n)
		wg      sync.WaitGroup
	)

	wg.Add(n)

	for i, url := range urls {
		go func(i int, url dataloader.Key) {
			defer wg.Done()

			resp, err := ldr.get.Film(ctx, url.String())
			results[i] = &dataloader.Result{Data: resp, Error: err}
		}(i, url)
	}

	wg.Wait()

	return results
}
