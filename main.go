package certCache

import (
	"context"
	"io/ioutil"
	"log"
	"strings"

	"cloud.google.com/go/storage"
	"golang.org/x/crypto/acme/autocert"
)

type StorageCache struct {
	bucket *storage.BucketHandle
}

func (sc StorageCache) Get(ctx context.Context, key string) ([]byte, error) {
	log.Printf("INFO certCache Fetching %+v from cache", key)
	contents := []byte{}

	obj := sc.bucket.Object(key)

	r, err := obj.NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return contents, autocert.ErrCacheMiss
		}
		log.Printf("ERROR certCache Fetching %+v from cache: %+v", key, err)
		return contents, err
	}
	defer r.Close()

	contents, err = ioutil.ReadAll(r)
	if err != nil {
		log.Printf("ERROR certCache Reading %+v from cache: %+v", key, err)
		return contents, err
	}

	return contents, nil
}

func (sc StorageCache) Put(ctx context.Context, key string, data []byte) error {
	log.Printf("INFO certCache Putting %+v into cache", key)
	obj := sc.bucket.Object(key)

	w := obj.NewWriter(ctx)
	if _, err := w.Write(data); err != nil {
		log.Printf("ERROR certCache Writing %+v to cache: %+v", key, err)
		return err
	}

	// Close, just like writing a file.
	if err := w.Close(); err != nil {
		log.Printf("ERROR certCache Closing %+v in cache: %+v", key, err)
		return err
	}

	return nil
}

func (sc StorageCache) Delete(ctx context.Context, key string) error {
	log.Printf("INFO certCache Deleting %+v from cache", key)
	obj := sc.bucket.Object(key)
	return obj.Delete(ctx)
}

func Init(bucketName, project string) (StorageCache, error) {
	ctx := context.Background()

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("ERROR certCache Failed to create client: %v", err)
	}

	sc := StorageCache{
		bucket: client.Bucket(bucketName),
	}

	if err := sc.bucket.Create(ctx, project, nil); err != nil {
		// Error responses described here https://cloud.google.com/storage/docs/json_api/v1/status-codes?hl=en
		// (note however that the docs as of 2022-02-17 are inaccurate/incomplete concerning the specific error message we should expect)
		// The two code 409 errors below are expected/acceptable (bucket already exists):
		// "Your previous request to create the named bucket succeeded and you already own it."
		// "You already own this bucket. Please select another name."
		if !strings.Contains(err.Error(), "ou already own") {
			return sc, err
		}
	} else {
		log.Printf("INFO certCache Certs bucket created.")
	}

	return sc, nil
}
