package main

import "fmt"

func main() {
	t := true
	f := false

	db := newDatabase("adrian-memory-db.rwqrpz.clustercfg.memorydb.us-east-1.amazonaws.com:6379", "sf-auth")
	db.users().insert(db.ctx(), &User{
		IDMOrgID:          8675432,
		IDMGUID:           "a5b36db4-e443-4a05-90f4-2a813d27b938",
		CRMOrganizationID: "00D1a000000KlceEAC",
		CRMUserID:         "0051a000000ZYEPAA4",
		Access:            &t,
		Sandbox:           &f,
		APIInstanceURL:    "https://na90.salesforce.com",
	})
	//usage: https://github.com/trilogy-group/xant-crm-sf-auth-adaptor-v2/blob/master/database/getUser.go#L31
	user1 := db.users().findOneByIdmGuid(db.ctx(), "a5b36db4-e443-4a05-90f4-2a813d27b938")
	fmt.Println(user1)
	user2 := db.users().findOneByIdmGuid(db.ctx(), "bla bla bla")
	fmt.Println(user2 == nil) //true
}
