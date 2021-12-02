package main

import (
	"context"
	"github.com/go-redis/redis/v8"
	"strconv"
	"strings"
	"time"
)

//https://github.com/trilogy-group/xant-crm-sf-auth-adaptor-v2/blob/master/database/mongo.go#L33
type Database struct {
	rdb       *redis.Client
	name      string
	opTimeout int
}

//mapping mongo.Collection
type Collection struct {
	database *Database
	name     string
}

//https://github.com/trilogy-group/xant-crm-sf-auth-adaptor-v2/blob/master/database/mongo.go#L287
type UserCollection struct {
	Collection
}

type TokenCollection struct {
	Collection
}

//https://github.com/trilogy-group/xant-crm-sf-auth-adaptor-v2/blob/master/database/mongo.go#L44
type Token struct {
	CRMOrganizationID string `bson:"crm_organization_id,omitempty"`
	CRMUserID         string `bson:"crm_user_id,omitempty"`
	Token             string `bson:"token,omitempty"`
	EncryptedToken    string `bson:"encrypted_token,omitempty"`
	FailureCount      *int   `bson:"failure_count,omitempty"`
}

//https://github.com/trilogy-group/xant-crm-sf-auth-adaptor-v2/blob/master/database/mongo.go#L53
type User struct {
	IDMOrgID          int64  `bson:"idm_org_id,omitempty"`
	IDMGUID           string `bson:"idm_guid,omitempty"`
	CRMOrganizationID string `bson:"crm_organization_id,omitempty"`
	CRMUserID         string `bson:"crm_user_id,omitempty"`
	Access            *bool  `bson:"access,omitempty"`
	Sandbox           *bool  `bson:"sandbox,omitempty"`
	APIInstanceURL    string `bson:"api_instance_url,omitempty"`
}

//check https://github.com/trilogy-group/xant-crm-sf-auth-adaptor-v2/blob/master/database/mongo.go#L146
func newDatabase(uri string, database string) *Database {
	rdb := redis.NewClient(&redis.Options{
		Addr:     uri,
		Password: "",
		DB:       0,
	})
	return &Database{
		rdb:       rdb,
		name:      database,
		opTimeout: 10,
	}
}

//https://github.com/trilogy-group/xant-crm-sf-auth-adaptor-v2/blob/master/database/mongo.go#L181
func (db *Database) ctx() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(db.opTimeout)*time.Second)
	return ctx
}

//https://github.com/trilogy-group/xant-crm-sf-auth-adaptor-v2/blob/master/database/mongo.go#L287
func (db *Database) users() *UserCollection {
	return &UserCollection{
		Collection: Collection{
			database: db,
			name:     "users",
		},
	}
}

//https://github.com/trilogy-group/xant-crm-sf-auth-adaptor-v2/blob/master/database/mongo.go#L279
func (db *Database) tokenCache() *TokenCollection {
	return &TokenCollection{
		Collection: Collection{
			database: db,
			name:     "token_cache",
		},
	}
}

//https://github.com/trilogy-group/xant-crm-sf-auth-adaptor-v2/blob/master/database/mongo.go#L283
func (db *Database) refreshTokens() *TokenCollection {
	return &TokenCollection{
		Collection: Collection{
			database: db,
			name:     "refresh_tokens",
		},
	}
}

func (c *Collection) key() string {
	return "crm." + c.database.name + "." + c.name
}

//put users in hset with keys crm.sf-auth.users.<some id>
func (uc *UserCollection) userKey(id string) string {
	return uc.key() + "." + id
}

//index users by idmguid in zset crm.sf-auth.users.idmguid.index
func (uc *UserCollection) idmGuidIndexKey() string {
	return uc.key() + ".idmguid.index"
}

func (uc *UserCollection) insert(ctx context.Context, user *User) {
	//put all fields in crm.sf-auth.users.<CRMUserID>
	uc.database.rdb.HSet(ctx, uc.userKey(user.CRMUserID),
		"idm_guid", user.IDMGUID,
		"idm_org_id", user.IDMOrgID,
		"crm_organization_id", user.CRMOrganizationID,
		"crm_user_id", user.CRMUserID,
		"access", *user.Access,
		"sandbox", *user.Sandbox,
		"api_instance_url", user.APIInstanceURL,
	)
	//index by idmguid by adding <IDMGUID:CRMUserID> in crm.sf-auth.users.idmguid.index
	uc.database.rdb.ZAdd(ctx, uc.idmGuidIndexKey(), &redis.Z{
		Score:  0,
		Member: user.IDMGUID + ":" + user.CRMUserID,
	})
}

func (uc *UserCollection) findOneByCrmUserId(ctx context.Context, CRMUserID string) *User {
	//get all fields from crm.sf-auth.users.<CRMUserID>
	fields := uc.database.rdb.HGetAll(ctx, uc.userKey(CRMUserID))

	t := true
	f := false

	asBool := func(val string) *bool {
		if val == "1" {
			return &t
		} else {
			return &f
		}
	}
	IDMOrgID, _ := strconv.ParseInt(fields.Val()["idm_org_id"], 10, 0)

	return &User{
		IDMOrgID:          IDMOrgID,
		IDMGUID:           fields.Val()["idm_guid"],
		CRMOrganizationID: fields.Val()["crm_organization_id"],
		CRMUserID:         fields.Val()["crm_user_id"],
		Access:            asBool(fields.Val()["access"]),
		Sandbox:           asBool(fields.Val()["sandbox"]),
		APIInstanceURL:    fields.Val()["api_instance_url"],
	}
}

func (uc *UserCollection) findOneByIdmGuid(ctx context.Context, IDMGUID string) *User {
	//search for IDMGUID:* in crm.sf-auth.users.idmguid.index
	idmGuidKey := "crm." + uc.database.name + "." + uc.name + ".idmguid.index"
	res := uc.database.rdb.ZRangeByLex(ctx, idmGuidKey, &redis.ZRangeBy{
		Min: "[" + IDMGUID,
		Max: "+",
	})

	if len(res.Val()) == 0 {
		return nil
	}

	//found a pair IDMGUID:CRMUserID, get the CRMUserID
	CRMUserID := strings.Split(res.Val()[0], ":")[1]

	return uc.findOneByCrmUserId(ctx, CRMUserID)
}
