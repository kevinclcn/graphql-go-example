package loader

import (
	"context"
	"sync"

	"github.com/nicksrandall/dataloader"

	"github.com/kevinclcn/graphql-go-example/errors"
	"github.com/kevinclcn/graphql-go-example/swapi"
)

func LoadPlanet(ctx context.Context, url string) (swapi.Planet, error) {
	var planet swapi.Planet

	ldr, err := extract(ctx, planetLoaderKey)
	if err != nil {
		return planet, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(url))()
	if err != nil {
		return planet, err
	}

	planet, ok := data.(swapi.Planet)
	if !ok {
		return planet, errors.WrongType(planet, data)
	}

	return planet, nil
}

func LoadPlanets(ctx context.Context, urls []string) (PlanetResults, error) {
	var results []PlanetResult

	ldr, err := extract(ctx, planetLoaderKey)
	if err != nil {
		return results, err
	}

	data, errs := ldr.LoadMany(ctx, dataloader.NewKeysFromStrings(urls))()
	results = make([]PlanetResult, 0, len(urls))

	for i, d := range data {
		var e error
		if errs != nil {
			e = errs[i]
		}

		planet, ok := d.(swapi.Planet)
		if !ok && e == nil {
			e = errors.WrongType(planet, d)
		}

		results = append(results, PlanetResult{Planet: planet, Error: e})
	}

	return results, nil
}

type PlanetResult struct {
	Planet swapi.Planet
	Error  error
}

type PlanetResults []PlanetResult

func (results PlanetResults) WithoutErrors() []swapi.Planet {
	planets := make([]swapi.Planet, 0, len(results))

	for _, r := range results {
		if r.Error != nil {
			continue
		}

		planets = append(planets, r.Planet)
	}

	return planets
}

func PrimePlanets(ctx context.Context, page swapi.PlanetPage) error {
	ldr, err := extract(ctx, planetLoaderKey)
	if err != nil {
		return err
	}

	for _, p := range page.Planets {
		ldr.Prime(ctx, dataloader.StringKey(p.URL), p)
	}
	return nil
}

type planetGetter interface {
	Planet(ctx context.Context, url string) (swapi.Planet, error)
}

type planetLoader struct {
	get planetGetter
}

func newPlanetLoader(client planetGetter) dataloader.BatchFunc {
	return planetLoader{get: client}.loadBatch
}

func (ldr planetLoader) loadBatch(ctx context.Context, urls dataloader.Keys) []*dataloader.Result {
	var (
		n       = len(urls)
		results = make([]*dataloader.Result, n)
		wg      sync.WaitGroup
	)

	wg.Add(n)

	for i, url := range urls {
		go func(i int, url dataloader.Key) {
			defer wg.Done()

			data, err := ldr.get.Planet(ctx, url.String())
			results[i] = &dataloader.Result{Data: data, Error: err}
		}(i, url)
	}

	wg.Wait()

	return results
}
