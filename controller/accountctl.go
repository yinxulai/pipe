// Pipe - A small and beautiful blogging platform written in golang.
// Copyright (C) 2017-2018, b3log.org
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package controller

import (
	"bytes"
	"context"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/yinxulai/pipe/model"
	"github.com/yinxulai/pipe/service"
	"github.com/yinxulai/pipe/util"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/qiniu/api.v7/storage"
	"github.com/satori/go.uuid"
	"github.com/tredoe/osutil/user/crypt/sha512_crypt"
)

// loginAction login a user.
func loginAction(c *gin.Context) {
	result := util.NewResult()
	defer c.JSON(http.StatusOK, result)

	arg := map[string]interface{}{}
	if err := c.BindJSON(&arg); nil != err {
		result.Code = -1
		result.Msg = "parses login request failed"

		return
	}

	name := arg["name"].(string)
	password := arg["password"].(string)

	user := service.User.GetUserByName(name)
	if nil == user {
		result.Code = -1
		result.Msg = "user not found"

		return
	}

	crypt := sha512_crypt.New()
	inputHash, _ := crypt.Generate([]byte(password), []byte(user.Password))
	if inputHash != user.Password {
		result.Code = -1
		result.Msg = "wrong password"

		return
	}

	ownBlog := service.User.GetOwnBlog(user.ID)
	session := &util.SessionData{
		UID:     user.ID,
		UName:   user.Name,
		UB3Key:  user.B3Key,
		UAvatar: user.AvatarURL,
		URole:   ownBlog.UserRole,
		BID:     ownBlog.ID,
		BURL:    ownBlog.URL,
	}
	if err := session.Save(c); nil != err {
		result.Code = -1
		result.Msg = "saves session failed: " + err.Error()
	}
}

// logoutAction logout a user.
func logoutAction(c *gin.Context) {
	result := util.NewResult()
	defer c.JSON(http.StatusOK, result)

	session := sessions.Default(c)
	session.Options(sessions.Options{
		Path:   "/",
		MaxAge: -1,
	})
	session.Clear()
	if err := session.Save(); nil != err {
		logger.Errorf("saves session failed: " + err.Error())
	}
}

// registerAction registers a user.
func registerAction(c *gin.Context) {
	result := util.NewResult()
	defer c.JSON(http.StatusOK, result)

	arg := map[string]interface{}{}
	if err := c.BindJSON(&arg); nil != err {
		result.Code = -1
		result.Msg = "parses register request failed"

		return
	}

	if !model.Conf.OpenRegister {
		result.Code = -1
		result.Msg = "Not open register at present"

		return
	}

	name := arg["name"].(string)
	password := arg["password"].(string)

	existUser := service.User.GetUserByName(name)
	if nil != existUser {
		result.Code = -1
		result.Msg = "duplicated user name"

		return
	}

	avatarURL := "https://img.hacpai.com/pipe/default-avatar.png"

	platformAdmin := service.User.GetPlatformAdmin()
	key := "pipe/" + platformAdmin.Name + "/" + name + "/" + name + "/" + strings.Replace(uuid.NewV4().String(), "-", "", -1) + ".jpg"
	avatarData := util.RandAvatarData()
	if nil != avatarData {
		uploadRet := &storage.PutRet{}
		refreshUploadToken()
		if err := storage.NewFormUploader(nil).Put(context.Background(), uploadRet, ut.token, key, bytes.NewReader(avatarData), int64(len(avatarData)), nil); nil != err {
			logger.Warnf("upload avatar to storage failed [" + err.Error() + "], uses default avatar instead")
		} else {
			avatarURL = ut.domain + "/" + uploadRet.Key
		}
	}

	user := &model.User{
		Name:      name,
		Password:  password,
		AvatarURL: avatarURL,
	}

	if err := service.Init.InitBlog(user); nil != err {
		result.Code = -1
		result.Msg = err.Error()

		return
	}

	ownBlog := service.User.GetOwnBlog(user.ID)
	session := &util.SessionData{
		UID:     user.ID,
		UName:   user.Name,
		UB3Key:  user.B3Key,
		UAvatar: user.AvatarURL,
		URole:   ownBlog.UserRole,
		BID:     ownBlog.ID,
		BURL:    ownBlog.URL,
	}
	if err := session.Save(c); nil != err {
		result.Code = -1
		result.Msg = "saves session failed: " + err.Error()
	}
}

func showLoginPageAction(c *gin.Context) {
	t, err := template.ParseFiles(filepath.ToSlash(filepath.Join(model.Conf.StaticRoot, "console/dist/login/index.html")))
	if nil != err {
		logger.Errorf("load login page failed: " + err.Error())
		c.String(http.StatusNotFound, "load login page failed")

		return
	}

	t.Execute(c.Writer, nil)
}

func showRegisterPageAction(c *gin.Context) {
	t, err := template.ParseFiles(filepath.ToSlash(filepath.Join(model.Conf.StaticRoot, "console/dist/register/index.html")))
	if nil != err {
		logger.Errorf("load register page failed: " + err.Error())
		c.String(http.StatusNotFound, "load register page failed")

		return
	}

	t.Execute(c.Writer, nil)
}
