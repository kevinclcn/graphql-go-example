package loader

import (
	"context"
	"sync"

	"github.com/nicksrandall/dataloader"

	"github.com/kevinclcn/graphql-go-example/errors"
	"github.com/kevinclcn/graphql-go-example/swapi"
)

// LoadPerson loads a person resource from the SWAPI API URL.
func LoadPerson(ctx context.Context, url string) (swapi.Person, error) {
	var person swapi.Person

	ldr, err := extract(ctx, personLoaderKey)
	if err != nil {
		return person, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(url))()
	if err != nil {
		return person, err
	}

	person, ok := data.(swapi.Person)
	if !ok {
		return person, errors.WrongType(person, data)
	}

	return person, nil
}

func LoadPeople(ctx context.Context, urls []string) (PersonResults, error) {
	var results []PersonResult

	ldr, err := extract(ctx, personLoaderKey)
	if err != nil {
		return results, err
	}

	data, errs := ldr.LoadMany(ctx, dataloader.NewKeysFromStrings(urls))()
	for i, d := range data {
		var e error
		if errs != nil {
			e = errs[i]
		}

		person, ok := d.(swapi.Person)
		if !ok && e == nil {
			e = errors.WrongType(person, d)
		}

		results = append(results, PersonResult{Person: person, Error: e})
	}

	return results, nil
}

type PersonResult struct {
	Person swapi.Person
	Error  error
}

type PersonResults []PersonResult

func (results PersonResults) WithoutErrors() []swapi.Person {
	people := make([]swapi.Person, 0, len(results))

	for _, r := range results {
		if r.Error != nil {
			continue
		}

		people = append(people, r.Person)
	}

	return people
}

func PrimePeople(ctx context.Context, page swapi.PersonPage) error {
	ldr, err := extract(ctx, personLoaderKey)
	if err != nil {
		return err
	}

	for _, p := range page.People {
		ldr.Prime(ctx, dataloader.StringKey(p.URL), p)
	}

	return nil
}

type personGetter interface {
	Person(ctx context.Context, url string) (swapi.Person, error)
}

// PersonLoader contains the RPC client necessary to load people.
type personLoader struct {
	get personGetter
}

func newPersonLoader(client personGetter) dataloader.BatchFunc {
	return personLoader{get: client}.loadBatch
}

func (ldr personLoader) loadBatch(ctx context.Context, urls dataloader.Keys) []*dataloader.Result {
	var (
		n       = len(urls)
		results = make([]*dataloader.Result, n)
		wg      sync.WaitGroup
	)

	wg.Add(n)

	for i, url := range urls {
		go func(i int, url dataloader.Key) {
			defer wg.Done()

			data, err := ldr.get.Person(ctx, url.String())
			results[i] = &dataloader.Result{Data: data, Error: err}
		}(i, url)
	}

	wg.Wait()

	return results
}
