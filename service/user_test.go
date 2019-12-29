package service

import (
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"math/rand"
	"testing"
	"time"
)

func TestCreateUser(t *testing.T) {
	user := &User{
		ID:        primitive.NewObjectID(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john" + randomString(20) + "@doe.com",
		Password:  "password",
	}
	userID, err := CreateUser(user)
	assert.NoError(t, err, "user creation failed")
	assert.NotEqual(t, primitive.ObjectID{}.Hex(), userID.(primitive.ObjectID).Hex(), "default object ID")
}

func TestGetUserByEmail(t *testing.T) {
	user := &User{
		ID:        primitive.NewObjectID(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john" + randomString(20) + "@doe.com",
		Password:  "password",
	}
	userID, err := CreateUser(user)
	assert.NoError(t, err, "user creation failed")
	assert.NotEqual(t, primitive.ObjectID{}.Hex(), userID.(primitive.ObjectID).Hex(), "default object ID")

	userFetched, err := GetUserByEmail(user.Email)
	assert.NoError(t, err, "user fetch failed")
	assert.Equal(t, userID.(primitive.ObjectID).Hex(), userFetched.ID.Hex(), "ID should be same")
	assert.Equal(t, user.FirstName, userFetched.FirstName, "first name should be same")
	assert.Equal(t, user.LastName, userFetched.LastName, "last name should be same")
	assert.Equal(t, user.Email, userFetched.Email, "email should be same")
	assert.NotEqual(t, primitive.ObjectID{}.Hex(), userFetched.InvitesId.Hex(), "invite ID should be created")
}

func TestGetUserByID(t *testing.T) {
	user := &User{
		ID:        primitive.NewObjectID(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john" + randomString(20) + "@doe.com",
		Password:  "password",
	}
	userID, err := CreateUser(user)
	assert.NoError(t, err, "user creation failed")
	assert.NotEqual(t, primitive.ObjectID{}.Hex(), userID.(primitive.ObjectID).Hex(), "default object ID")

	userFetched, err := GetUserByID(userID.(primitive.ObjectID))
	assert.NoError(t, err, "user fetch failed")
	assert.Equal(t, userID.(primitive.ObjectID).Hex(), userFetched.ID.Hex(), "ID should be same")
	assert.Equal(t, user.FirstName, userFetched.FirstName, "first name should be same")
	assert.Equal(t, user.LastName, userFetched.LastName, "last name should be same")
	assert.Equal(t, user.Email, userFetched.Email, "email should be same")
	assert.NotEqual(t, primitive.ObjectID{}.Hex(), userFetched.InvitesId.Hex(), "invite ID should be created")
}

func TestUser_LoginUser(t *testing.T) {
	user := &User{
		ID:        primitive.NewObjectID(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john" + randomString(20) + "@doe.com",
		Password:  "password",
	}
	userID, err := CreateUser(user)
	assert.NoError(t, err, "user creation failed")
	assert.NotEqual(t, primitive.ObjectID{}.Hex(), userID.(primitive.ObjectID).Hex(), "default object ID")

	lastLogin, err := user.LoginUser("password")
	assert.NoError(t, err, "user login failed")
	assert.True(t, len(lastLogin) > 0, "last login will be some non-empty timestamp")
}

func TestUser_ExistingUser(t *testing.T) {
	user := &User{
		ID:        primitive.NewObjectID(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john" + randomString(20) + "@doe.com",
		Password:  "password",
	}
	userID, err := CreateUser(user)
	assert.NoError(t, err, "user creation failed")
	assert.NotEqual(t, primitive.ObjectID{}.Hex(), userID.(primitive.ObjectID).Hex(), "default object ID")

	exists := user.ExistingUser()
	assert.NoError(t, err, "existing user check failed")
	assert.True(t, exists, "user should exist as just created")
}

func TestUser_UpdatePassword(t *testing.T) {
	user := &User{
		ID:        primitive.NewObjectID(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john" + randomString(20) + "@doe.com",
		Password:  "password",
	}
	userID, err := CreateUser(user)
	assert.NoError(t, err, "user creation failed")
	assert.NotEqual(t, primitive.ObjectID{}.Hex(), userID.(primitive.ObjectID).Hex(), "default object ID")

	hash, err := bcrypt.GenerateFromPassword([]byte("password_new"), bcrypt.DefaultCost)
	assert.NoError(t, err, "password hashing failed")

	err = user.UpdatePassword(string(hash))
	assert.NoError(t, err, "update password failed")
}

func TestUser_UpdateName(t *testing.T) {
	user := &User{
		ID:        primitive.NewObjectID(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john" + randomString(20) + "@doe.com",
		Password:  "password",
	}
	userID, err := CreateUser(user)
	assert.NoError(t, err, "user creation failed")
	assert.NotEqual(t, primitive.ObjectID{}.Hex(), userID.(primitive.ObjectID).Hex(), "default object ID")

	err = user.UpdateName("firstName", "lastName")
	assert.NoError(t, err, "update name failed")
}

func TestUser_SeeOnlineFriends(t *testing.T) {
	user := &User{
		ID:        primitive.NewObjectID(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john" + randomString(20) + "@doe.com",
		Password:  "password",
	}
	userID, err := CreateUser(user)
	assert.NoError(t, err, "user creation failed")
	assert.NotEqual(t, primitive.ObjectID{}.Hex(), userID.(primitive.ObjectID).Hex(), "default object ID")

	friends, err := user.SeeOnlineFriends()
	assert.NoError(t, err, "checking online friends failed")
	assert.Equal(t, 0, len(friends), "as no friends for the current user")
}

func TestUser_Logout(t *testing.T) {
	user := &User{
		ID:        primitive.NewObjectID(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john" + randomString(20) + "@doe.com",
		Password:  "password",
	}
	userID, err := CreateUser(user)
	assert.NoError(t, err, "user creation failed")
	assert.NotEqual(t, primitive.ObjectID{}.Hex(), userID.(primitive.ObjectID).Hex(), "default object ID")

	lastLogin, err := user.LoginUser("password")
	assert.NoError(t, err, "user login failed")
	assert.True(t, len(lastLogin) > 0, "last login will be some non-empty timestamp")

	err = user.Logout()
	assert.NoError(t, err, "user logout failed")
}

func TestUser_SendInvitation(t *testing.T) {
	user1 := new(User)
	user1.FirstName = "John"
	user1.LastName = "Doe"
	user1.Email = "john" + randomString(15) + "@doe.com"
	user1.Password = "password"
	_, err := CreateUser(user1)
	assert.NoError(t, err, "user creation failed")

	user2 := new(User)
	user2.FirstName = "John2"
	user2.LastName = "Doe2"
	user2.Email = "john" + randomString(15) + "@doe.com"
	user2.Password = "password"
	user2ID, err := CreateUser(user2)
	assert.NoError(t, err, "user creation failed")

	user2.ID = user2ID.(primitive.ObjectID)
	err = user1.SendInvitation(user2)
	assert.NoError(t, err, "send new user invitations failed")
}

func TestUser_AddFriend(t *testing.T) {
	user1 := new(User)
	user1.FirstName = "John"
	user1.LastName = "Doe"
	user1.Email = "john" + randomString(15) + "@doe.com"
	user1.Password = "password"
	user1ID, err := CreateUser(user1)
	assert.NoError(t, err, "user creation failed")

	user2 := new(User)
	user2.FirstName = "John2"
	user2.LastName = "Doe2"
	user2.Email = "john" + randomString(15) + "@doe.com"
	user2.Password = "password"
	user2ID, err := CreateUser(user2)
	assert.NoError(t, err, "user creation failed")

	user2.ID = user2ID.(primitive.ObjectID)
	err = user1.SendInvitation(user2)
	assert.NoError(t, err, "sending new user invitation failed")

	user2.ID = user2ID.(primitive.ObjectID)
	err = user2.AddFriend(user1ID.(primitive.ObjectID))
	assert.NoError(t, err, "adding new friend failed")
}

func TestValidUserEmail(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{
			name:  "valid email",
			input: "john@doe.com",
			valid: true,
		},
		{
			name:  "another valid email",
			input: "john1@doe.co.in",
			valid: true,
		},
		{
			name:  "invalid email",
			input: "john1doe.co.in",
			valid: false,
		},
		{
			name:  "only domain email",
			input: "gmail.com",
			valid: false,
		},
		{
			name:  "only extension invalid email",
			input: ".com",
			valid: false,
		},
		{
			name:  "empty email",
			input: "",
			valid: false,
		},
	}
	for _, tc := range tests {
		valid := ValidUserEmail(tc.input)
		assert.Equal(t, tc.valid, valid, "email validation failed for test %s", tc.name)
	}
}

func TestUser_GetSentInvitations(t *testing.T) {
	user1 := new(User)
	user1.FirstName = "John"
	user1.LastName = "Doe"
	user1.Email = "john" + randomString(15) + "@doe.com"
	user1.Password = "password"
	_, err := CreateUser(user1)
	assert.NoError(t, err, "user creation failed")

	invites, err := user1.GetSentInvitations()
	assert.NoError(t, err, "fetching new user invitations failed")
	assert.Equal(t, 0, len(invites), "empty list of invites should come")
}

func TestUser_GetReceivedInvitations(t *testing.T) {
	user1 := new(User)
	user1.FirstName = "John"
	user1.LastName = "Doe"
	user1.Email = "john" + randomString(15) + "@doe.com"
	user1.Password = "password"
	_, err := CreateUser(user1)
	assert.NoError(t, err, "user creation failed")

	invites, err := user1.GetReceivedInvitations()
	assert.NoError(t, err, "fetching new user invitations failed")
	assert.Equal(t, 0, len(invites), "empty list of invites should come")
}

func TestUser_GetAcceptedInvitations(t *testing.T) {
	user1 := new(User)
	user1.FirstName = "John"
	user1.LastName = "Doe"
	user1.Email = "john" + randomString(15) + "@doe.com"
	user1.Password = "password"
	_, err := CreateUser(user1)
	assert.NoError(t, err, "user creation failed")

	invites, err := user1.GetAcceptedInvitations()
	assert.NoError(t, err, "fetching new user invitations failed")
	assert.Equal(t, 0, len(invites), "empty list of invites should come")
}

func TestUser_GetCanceledSentInvitations(t *testing.T) {
	user1 := new(User)
	user1.FirstName = "John"
	user1.LastName = "Doe"
	user1.Email = "john" + randomString(15) + "@doe.com"
	user1.Password = "password"
	_, err := CreateUser(user1)
	assert.NoError(t, err, "user creation failed")

	invites, err := user1.GetCanceledSentInvitations()
	assert.NoError(t, err, "fetching new user invitations failed")
	assert.Equal(t, 0, len(invites), "empty list of invites should come")
}

func TestUser_GetRejectedInvitations(t *testing.T) {
	user1 := new(User)
	user1.FirstName = "John"
	user1.LastName = "Doe"
	user1.Email = "john" + randomString(15) + "@doe.com"
	user1.Password = "password"
	_, err := CreateUser(user1)
	assert.NoError(t, err, "user creation failed")

	invites, err := user1.GetReceivedInvitations()
	assert.NoError(t, err, "fetching new user invitations failed")
	assert.Equal(t, 0, len(invites), "empty list of invites should come")
}

func TestUser_CancelInvitation(t *testing.T) {
	user1 := new(User)
	user1.FirstName = "John"
	user1.LastName = "Doe"
	user1.Email = "john" + randomString(15) + "@doe.com"
	user1.Password = "password"
	_, err := CreateUser(user1)
	assert.NoError(t, err, "user creation failed")

	user2 := new(User)
	user2.FirstName = "John2"
	user2.LastName = "Doe2"
	user2.Email = "john" + randomString(15) + "@doe.com"
	user2.Password = "password"
	user2ID, err := CreateUser(user2)
	assert.NoError(t, err, "user creation failed")

	user2.ID = user2ID.(primitive.ObjectID)
	err = user1.SendInvitation(user2)
	assert.NoError(t, err, "fetching new user invitations failed")

	user2.ID = user2ID.(primitive.ObjectID)
	err = user1.CancelInvitation(user2)
	assert.NoError(t, err, "fetching new user invitations failed")
}

func TestUser_SeeFriends(t *testing.T) {
	user := &User{
		ID:        primitive.NewObjectID(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john" + randomString(20) + "@doe.com",
		Password:  "password",
	}
	userID, err := CreateUser(user)
	assert.NoError(t, err, "user creation failed")
	assert.NotEqual(t, primitive.ObjectID{}.Hex(), userID.(primitive.ObjectID).Hex(), "default object ID")

	friends, err := user.SeeFriends()
	assert.NotNil(t, err, "as no friends for the user currently")
	assert.Equal(t, 0, len(friends), "as no friends for the current user")
}

func TestUserProfile(t *testing.T) {
	user := &User{
		ID:        primitive.NewObjectID(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john" + randomString(20) + "@doe.com",
		Password:  "password",
	}
	userID, err := CreateUser(user)
	assert.NoError(t, err, "user creation failed")
	assert.NotEqual(t, primitive.ObjectID{}.Hex(), userID.(primitive.ObjectID).Hex(), "default object ID")

	userProfile, err := UserProfile(userID.(primitive.ObjectID))
	assert.NoError(t, err, "user profile fetch failed")
	assert.NotEqual(t, 0, len(userProfile), "user profile fetch failed")
}

func TestUser_String(t *testing.T) {
	user := &User{
		ID: primitive.NewObjectID(),
	}
	assert.Equal(t, len(primitive.ObjectID{}.String()), len(user.String()))
	assert.NotEqual(t, primitive.ObjectID{}.String(), user.String())
}

func TestUser_ShowChat(t *testing.T) {
	// fetching a genuine chat, so creating users and chat b/w them
	user1 := new(User)
	user1.FirstName = "John"
	user1.LastName = "Doe"
	user1.Email = "john" + randomString(15) + "@doe.com"
	user1.Password = "password"
	user1ID, err := CreateUser(user1)
	assert.NoError(t, err, "user creation failed")

	user2 := new(User)
	user2.FirstName = "John2"
	user2.LastName = "Doe2"
	user2.Email = "john" + randomString(15) + "@doe.com"
	user2.Password = "password"
	user2ID, err := CreateUser(user2)
	assert.NoError(t, err, "user creation failed")

	err = SendMessage(user1ID.(primitive.ObjectID), user2ID.(primitive.ObjectID), "test message")
	assert.NoError(t, err, "new document should be created for the chat")

	content, timestamp := user1.ShowChat(user2ID.(primitive.ObjectID))
	assert.NotEqual(t, "", content, "non-empty content should arrive")
	assert.NotEqual(t, time.Time{}, timestamp, "non-empty (non-ZERO value) should be returned")
}

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func randomString(length int) string {
	b := make([]byte, length)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := length-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return string(b)
}
