package testhelpers

import (
	"context"

	"cloud.google.com/go/storage"
	. "github.com/onsi/gomega"
)

func DeleteGCSBucket(ctx context.Context, bucket *storage.BucketHandle) {
	objectsInBucket := bucket.Objects(ctx, nil)
	for {
		obj, err := objectsInBucket.Next()
		if err == storage.Done {
			break
		}
		Expect(err).NotTo(HaveOccurred())
		Expect(bucket.Object(obj.Name).Delete(ctx)).To(Succeed())
	}
	Expect(bucket.Delete(ctx)).To(Succeed())
}
