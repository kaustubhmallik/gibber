package service

import (
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
)

func TestCreateUser(t *testing.T) {
	user := &User{
		ID:        primitive.NewObjectID(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john" + RandomString(20) + "@doe.com",
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
		Email:     "john" + RandomString(20) + "@doe.com",
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
		Email:     "john" + RandomString(20) + "@doe.com",
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
		Email:     "john" + RandomString(20) + "@doe.com",
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
		Email:     "john" + RandomString(20) + "@doe.com",
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
		Email:     "john" + RandomString(20) + "@doe.com",
		Password:  "password",
	}
	userID, err := CreateUser(user)
	assert.NoError(t, err, "user creation failed")
	assert.NotEqual(t, primitive.ObjectID{}.Hex(), userID.(primitive.ObjectID).Hex(), "default object ID")

	hash, err := GenerateHash("password_new")
	assert.NoError(t, err, "password hashing failed")

	err = user.UpdatePassword(hash)
	assert.NoError(t, err, "update password failed")
}

func TestUser_UpdateName(t *testing.T) {
	user := &User{
		ID:        primitive.NewObjectID(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john" + RandomString(20) + "@doe.com",
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
		Email:     "john" + RandomString(20) + "@doe.com",
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
		Email:     "john" + RandomString(20) + "@doe.com",
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

}

func TestUser_AddFriend(t *testing.T) {

}

func TestUser_GetSentInvitations(t *testing.T) {

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

func TestUser_GetReceivedInvitations(t *testing.T) {

}

func TestUser_GetAcceptedInvitations(t *testing.T) {

}

func TestUser_GetCanceledSentInvitations(t *testing.T) {

}

func TestUser_CancelInvitation(t *testing.T) {

}

func TestUser_SeeFriends(t *testing.T) {
	user := &User{
		ID:        primitive.NewObjectID(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john" + RandomString(20) + "@doe.com",
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
		Email:     "john" + RandomString(20) + "@doe.com",
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

}
