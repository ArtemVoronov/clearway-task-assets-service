package services

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	createAssetQuery = `INSERT INTO assets (name, user_uuid, file_id) VALUES ($1, $2, $3)`
	getAssetQuery    = `SELECT file_id FROM assets WHERE name = $1`
)

var ErrDuplicateAsset = errors.New("duplicate asset")
var ErrNotFoundAsset = errors.New("asset not found")

type AssetsService struct {
	clientShards []*PostgreSQLService
	ShardsNum    int
	shardService *ShardService
}

func CreateAssetsService(clients []*PostgreSQLService) *AssetsService {
	return &AssetsService{
		clientShards: clients,
		ShardsNum:    len(clients),
		shardService: CreateShardService(len(clients)),
	}
}

func (s *AssetsService) Shutdown() error {
	return nil
}

func (s *AssetsService) client(userUuid string) *PostgreSQLService {
	bucketIndex := s.shardService.GetBucketIndex(userUuid)
	bucket := s.shardService.GetBucketByIndex(bucketIndex)
	return s.clientShards[bucket]
}

func (s *AssetsService) Client(userUuid string) *PostgreSQLService {
	bucketIndex := s.shardService.GetBucketIndex(userUuid)
	bucket := s.shardService.GetBucketByIndex(bucketIndex)
	return s.clientShards[bucket]
}

func (s *AssetsService) CreateAsset(name string, userUuid string, file io.Reader) error {
	err := s.client(userUuid).TxVoid(
		func(tx pgx.Tx, ctx context.Context, cancel context.CancelFunc) error {
			lobs := tx.LargeObjects()

			oid, internalErr := lobs.Create(ctx, 0)
			if internalErr != nil {
				return internalErr
			}

			_, internalErr = tx.Exec(ctx, createAssetQuery, name, userUuid, oid)
			if internalErr != nil {
				var pgErr *pgconn.PgError
				switch {
				case errors.As(internalErr, &pgErr):
					if pgErr.Code == DuplicateErrorCode {
						return ErrDuplicateAsset
					} else {
						return fmt.Errorf("user '%v' unable to insert assert with name '%v': %w", userUuid, name, internalErr)
					}
				default:
					return fmt.Errorf("user '%v' unable to insert assert with name '%v': %w", userUuid, name, internalErr)
				}

			}

			obj, internalErr := lobs.Open(ctx, oid, pgx.LargeObjectModeWrite)
			if internalErr != nil {
				return internalErr
			}

			_, internalErr = io.Copy(obj, file)
			if internalErr != nil {
				return internalErr
			}
			return internalErr
		},
		pgx.TxOptions{
			IsoLevel: pgx.ReadCommitted,
		})()

	return err
}

type StartStreamingFunc func(content io.ReadSeeker)

func (s *AssetsService) GetAsset(name string, userUuid string, startStreaming StartStreamingFunc) error {
	err := s.client(userUuid).TxVoid(
		func(tx pgx.Tx, ctx context.Context, cancel context.CancelFunc) error {
			var oid uint32
			internalErr := tx.QueryRow(ctx, getAssetQuery, name).Scan(&oid)
			if internalErr != nil {
				return internalErr
			}

			lobs := tx.LargeObjects()
			obj, internalErr := lobs.Open(ctx, oid, pgx.LargeObjectModeRead)
			if internalErr != nil {
				return internalErr
			}

			startStreaming(obj)

			return nil
		},
		pgx.TxOptions{
			IsoLevel: pgx.ReadCommitted,
		})()

	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrNotFoundAsset
		}
		return fmt.Errorf("unable to get asset: %w", err)
	}

	return nil
}
