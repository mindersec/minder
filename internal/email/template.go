// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package email

//nolint:lll
const (
	// bodyHTML is the HTML body of the email
	bodyHTML = `
<div
  style="
    background-color: #f5fbff;
    color: #262626;
    font-family: 'Helvetica Neue', 'Arial Nova', 'Nimbus Sans', Arial,
      sans-serif;
    font-size: 16px;
    font-weight: 400;
    letter-spacing: 0.15008px;
    line-height: 1.5;
    margin: 0;
    padding: 32px 0;
    min-height: 100%;
    width: 100%;
  "
>
  <table
    align="center"
    width="100%"
    style="margin: 0 auto; max-width: 600px; background-color: #ffffff"
    cellspacing="0"
    cellpadding="0"
    border="0"
  >
    <tbody>
      <tr style="width: 100%">
        <td>
          <div
            style="
              padding: 20px 24px 20px 24px;
              background-color: #f5fbff;
              text-align: center;
            "
          >
            <img
              alt="Minder by Stacklok"
              src="https://stacklok-statamic-1.nyc3.digitaloceanspaces.com/email_minder_logo.png"
              width="134"
              height="32"
              style="
                width: 134px;
                height: 32px;
                outline: none;
                border: none;
                text-decoration: none;
                vertical-align: middle;
                max-width: 100%;
              "
            />
          </div>
          <div
            style="
              background-color: #ffffff;
              border-radius: 0;
              padding: 16px 32px 16px 32px;
            "
          >
            <div
              style="
                color: #475467;
                font-size: 14px;
                font-weight: normal;
                padding: 6px 0px 0px 0px;
              "
            >
              <p>
                <strong>{{.AdminName}}</strong> has invited you to become <strong>{{.RoleName}}</strong> in
                the <strong>{{.OrganizationName}}</strong> organization in Minder.
              </p>
            </div>
            <div style="text-align: center; padding: 4px 24px 8px 24px">
              <table align="center">
                <tr>
                  <td align="center">
                    <div>
                      <!--[if mso]>
  <v:rect xmlns:v="urn:schemas-microsoft-com:vml" xmlns:w="urn:schemas-microsoft-com:office:word" href="{{.InvitationURL}}" style="height:36px;v-text-anchor:middle;width:110px;" stroke="f" fillcolor="#803eff">
    <w:anchorlock/>
    <center>
  <![endif]-->
                      <a
                        href="{{.InvitationURL}}"
                        target="_blank"
                        style="
                          background-color: #803eff;
                          color: #ffffff;
                          display: block;
                          font-family: Helvetica, Arial, sans-serif;
                          font-size: 12px;
                          line-height: 36px;
                          height: 36px;
                          text-align: center;
                          font-weight: bold;
                          text-decoration: none;
                          width: 110px;
                          border-radius: 8px;
                          -webkit-text-size-adjust: none;
                        "
                        >View Invitation</a
                      >
                      <!--[if mso]>
    </center>
  </v:rect>
<![endif]-->
                    </div>
                  </td>
                </tr>
              </table>
            </div>
            <div
              style="
                color: #475467;
                font-size: 14px;
                font-weight: normal;
                padding: 18px 0px 6px 0px;
              "
            >
              Once you accept, you’ll be able to view the {{.OrganizationName}}
              organization in Minder.
            </div>
            <div
              style="
                color: #475467;
                font-size: 12px;
                font-weight: normal;
                padding: 6px 0px 6px 0px;
              "
            >
              <p>
                This invitation was sent to
                <strong>{{.RecipientEmail}}</strong>. If you were not
                expecting it, you can ignore this email.
              </p>
            </div>
            <div style="padding: 6px 0px 6px 0px">
              <hr
                style="
                  width: 100%;
                  border: none;
                  border-top: 1px solid #eaecf0;
                  margin: 0;
                "
              />
            </div>
            <div
              style="
                color: #475467;
                font-size: 12px;
                font-weight: normal;
                padding: 6px 0px 6px 0px;
              "
            >
              <p>
                <a href="{{.MinderURL}}" style="color: #6941c6">Minder</a>
                by Stacklok is an open source platform that helps development
                teams and open source communities build more secure software,
                and prove to others that what they’ve built is secure.
              </p>
            </div>
            <div style="padding: 8px 24px 8px 24px">
              <table
                align="center"
                width="100%"
                cellpadding="0"
                border="0"
                style="border-collapse: collapse"
              >
                <tbody style="width: 100%">
                  <tr style="width: 100%">
                    <td
                      style="
                        box-sizing: content-box;
                        vertical-align: middle;
                        padding-left: 0;
                        padding-right: 10.666666666666666px;
                      "
                    >
                      <div
                        style="
                          font-size: 12px;
                          font-weight: normal;
                          text-align: right;
                          padding: 0px 0px 0px 0px;
                        "
                      >
                        <p>
                          <a href="{{.TermsURL}}" style="color: #475467"
                            >Terms and Conditions</a
                          >
                        </p>
                      </div>
                    </td>
                    <td
                      style="
                        box-sizing: content-box;
                        vertical-align: middle;
                        padding-left: 5.333333333333333px;
                        padding-right: 5.333333333333333px;
                        width: 70px;
                      "
                    >
                      <div
                        style="
                          font-size: 12px;
                          font-weight: normal;
                          text-align: center;
                          padding: 0px 16px 0px 16px;
                        "
                      >
                        <p>
                          <a href="{{.PrivacyURL}}" style="color: #475467"
                            >Privacy</a
                          >
                        </p>
                      </div>
                    </td>
                    <td
                      style="
                        box-sizing: content-box;
                        vertical-align: middle;
                        padding-left: 10.666666666666666px;
                        padding-right: 0;
                      "
                    >
                      <div
                        style="
                          font-size: 12px;
                          font-weight: normal;
                          padding: 0px 0px 0px 0px;
                        "
                      >
                        <p>
                          <a href="{{.SignInURL}}" style="color: #475467"
                            >Sign in to Minder</a
                          >
                        </p>
                      </div>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
          <div
            style="
              padding: 28px 24px 40px 24px;
              background-color: #f5fbff;
              text-align: center;
            "
          >
            <img
              alt="Stacklok"
              src="https://stacklok-statamic-1.nyc3.digitaloceanspaces.com/email_stacklok_logo.png"
              height="20"
              style="
                height: 20px;
                outline: none;
                border: none;
                text-decoration: none;
                vertical-align: middle;
                max-width: 100%;
              "
            />
          </div>
        </td>
      </tr>
    </tbody>
  </table>
</div>
`
)
