package s3

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
)

const (
	deleteBatchSize = 500
)

// emptyBucket empties the specified S3 bucket by deleting all object versions and delete markers.
// If `force` is `true` then S3 Object Lock governance mode restrictions are bypassed and
// an attempt is made to remove any S3 Object Lock legal holds.
func emptyBucket(ctx context.Context, conn *s3.S3, bucket string, force bool) error {
	deleter := s3manager.NewBatchDeleteWithClient(conn, func(o *s3manager.BatchDelete) { o.BatchSize = deleteBatchSize })

	// First attempt to delete all object versions.
	objectVersionIterator := NewDeleteObjectVersionListIterator(conn, bucket, "", force)
	err := deleter.Delete(ctx, objectVersionIterator)

	if err != nil {
		if !force {
			return err
		}

		var batchErr *s3manager.BatchError

		if errors.As(err, &batchErr) {
			for _, v := range batchErr.Errors {
				if tfawserr.ErrCodeEquals(v.OrigErr, "AccessDenied") {

				}
			}
		}
	}

	return nil
}

// listIterator is intended to be embedded inside iterators.
type listIterator struct {
	bucket                    string
	bypassGovernanceRetention bool
	key                       string
	paginator                 request.Pagination
}

// deleteVersionListIterator implements s3manager.BatchDeleteIterator.
// It iterates through a list of S3 Object versions and delete them.
// It is inspired by s3manager.DeleteListIterator.
type deleteObjectVersionListIterator struct {
	listIterator
	objects []*s3.ObjectVersion
}

func NewDeleteObjectVersionListIterator(conn *s3.S3, bucket, key string, bypassGovernanceRetention bool) s3manager.BatchDeleteIterator {
	return &deleteObjectVersionListIterator{
		listIterator: listIterator{
			bucket:                    bucket,
			bypassGovernanceRetention: bypassGovernanceRetention,
			key:                       key,
			paginator:                 listObjectVersionsPaginator(conn, bucket, key),
		},
	}
}

func (iter *deleteObjectVersionListIterator) Next() bool {
	if len(iter.objects) > 0 {
		iter.objects = iter.objects[1:]
	}

	if len(iter.objects) == 0 && iter.listIterator.paginator.Next() {
		if iter.key == "" {
			iter.objects = iter.listIterator.paginator.Page().(*s3.ListObjectVersionsOutput).Versions
		} else {
			// ListObjectVersions uses Prefix as an argument but we use Key.
			// Ignore any object versions that do not have the required Key.
			for _, v := range iter.listIterator.paginator.Page().(*s3.ListObjectVersionsOutput).Versions {
				if iter.key != aws.StringValue(v.Key) {
					continue
				}

				iter.objects = append(iter.objects, v)
			}
		}
	}

	return len(iter.objects) > 0
}

func (iter *deleteObjectVersionListIterator) Err() error {
	return iter.listIterator.paginator.Err()
}

func (iter *deleteObjectVersionListIterator) DeleteObject() s3manager.BatchDeleteObject {
	return s3manager.BatchDeleteObject{
		Object: &s3.DeleteObjectInput{
			Bucket:                    aws.String(iter.listIterator.bucket),
			BypassGovernanceRetention: aws.Bool(iter.listIterator.bypassGovernanceRetention),
			Key:                       iter.objects[0].Key,
			VersionId:                 iter.objects[0].VersionId,
		},
	}
}

// deleteDeleteMarkerListIterator implements s3manager.BatchDeleteIterator.
// It iterates through a list of S3 Object delete markers and delete them.
// It is inspired by s3manager.DeleteListIterator.
type deleteDeleteMarkerListIterator struct {
	listIterator
	deleteMarkers []*s3.DeleteMarkerEntry
}

func NewDeleteDeleteMarkerListIterator(conn *s3.S3, bucket, key string, bypassGovernanceRetention bool) s3manager.BatchDeleteIterator {
	return &deleteDeleteMarkerListIterator{
		listIterator: listIterator{
			bucket:                    bucket,
			bypassGovernanceRetention: bypassGovernanceRetention,
			key:                       key,
			paginator:                 listObjectVersionsPaginator(conn, bucket, key),
		},
	}
}

func (iter *deleteDeleteMarkerListIterator) Next() bool {
	if len(iter.deleteMarkers) > 0 {
		iter.deleteMarkers = iter.deleteMarkers[1:]
	}

	if len(iter.deleteMarkers) == 0 && iter.listIterator.paginator.Next() {
		if iter.key == "" {
			iter.deleteMarkers = iter.listIterator.paginator.Page().(*s3.ListObjectVersionsOutput).DeleteMarkers
		} else {
			// ListObjectVersions uses Prefix as an argument but we use Key.
			// Ignore any delete markers that do not have the required Key.
			for _, v := range iter.listIterator.paginator.Page().(*s3.ListObjectVersionsOutput).DeleteMarkers {
				if iter.key != aws.StringValue(v.Key) {
					continue
				}

				iter.deleteMarkers = append(iter.deleteMarkers, v)
			}
		}
	}

	return len(iter.deleteMarkers) > 0
}

func (iter *deleteDeleteMarkerListIterator) Err() error {
	return iter.listIterator.paginator.Err()
}

func (iter *deleteDeleteMarkerListIterator) DeleteObject() s3manager.BatchDeleteObject {
	return s3manager.BatchDeleteObject{
		Object: &s3.DeleteObjectInput{
			Bucket:                    aws.String(iter.listIterator.bucket),
			BypassGovernanceRetention: aws.Bool(iter.listIterator.bypassGovernanceRetention),
			Key:                       iter.deleteMarkers[0].Key,
			VersionId:                 iter.deleteMarkers[0].VersionId,
		},
	}
}

// listObjectVersionsPaginator returns a paginator that lists S3 object versions for the specified bucket and optional key.
func listObjectVersionsPaginator(conn *s3.S3, bucket, key string) request.Pagination {
	return request.Pagination{
		NewRequest: func() (*request.Request, error) {
			input := &s3.ListObjectVersionsInput{
				Bucket: aws.String(bucket),
			}

			if key != "" {
				input.Prefix = aws.String(key)
			}

			request, _ := conn.ListObjectVersionsRequest(input)

			return request, nil
		},
	}
}
