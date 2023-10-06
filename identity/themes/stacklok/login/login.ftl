<#import "template.ftl" as layout>

<link href='https://fonts.googleapis.com/css?family=Figtree' rel='stylesheet'>

<div class="layout-simple">
    <div id="main" class="grid12">
        <div class="grid7">
            <div class="width-limited">
                <div class="stacklok-logo"></div>
                <div class="title">Sign In to<br>Unlock Additional Possibilities</div>
                <div class="subtitle">Welcome back!</div>
            </div>
           <div class="width-limited">
            <@layout.registrationLayout displayInfo=social.displayInfo; section>

            <#if section = "title">
                <#elseif section = "form">
                    <#if realm.password>

                        <form id="kc-form-login" class="${properties.kcFormClass!}" onsubmit="login.disabled = true; return true;" action="${url.loginAction}" method="post">


                            <div class="${properties.kcFormGroupClass!}">
                                <div class="${properties.kcLabelWrapperClass!}">
                                    <label for="username" class="${properties.kcLabelClass!} label"><#if !realm.loginWithEmailAllowed>${msg("username")}<#elseif !realm.registrationEmailAsUsername>${msg("usernameOrEmail")}<#else>${msg("email")}</#if></label>
                                </div>

                                <div class="${properties.kcInputWrapperClass!}">
                                    <#if usernameEditDisabled??>
                                        <input tabindex="1" id="username" class="${properties.kcInputClass!} input" name="username" value="${(login.username!'')}" type="text" disabled />
                                    <#else>
                                        <input tabindex="1" id="username" class="${properties.kcInputClass!} input" name="username" value="${(login.username!'')}" type="text" autofocus autocomplete="off" />
                                    </#if>
                                </div>
                            </div>

                            <div class="${properties.kcFormGroupClass!}">
                                <div class="${properties.kcLabelWrapperClass!}">
                                    <label for="password" class="${properties.kcLabelClass!} label">${msg("password")}</label>
                                </div>

                                <div class="${properties.kcInputWrapperClass!}">
                                    <input tabindex="2" id="password" class="${properties.kcInputClass!} input" name="password" type="password" autocomplete="off" />
                                </div>
                            </div>

                            <div class="${properties.kcFormGroupClass!}">
                                <div id="kc-form-options" class="${properties.kcFormOptionsClass!}">
                                    <#if realm.rememberMe && !usernameEditDisabled??>
                                        <div class="checkbox">
                                            <label>
                                                <#if login.rememberMe??>
                                                    <input tabindex="3" id="rememberMe" name="rememberMe" type="checkbox" tabindex="3" checked> ${msg("rememberMe")}
                                                <#else>
                                                    <input tabindex="3" id="rememberMe" name="rememberMe" type="checkbox" tabindex="3"> ${msg("rememberMe")}
                                                </#if>
                                            </label>
                                        </div>
                                    </#if>
                                    <div class="${properties.kcFormOptionsWrapperClass!}">
                                        <#if realm.resetPasswordAllowed>
                                            <span><a tabindex="5" href="${url.loginResetCredentialsUrl}">${msg("doForgotPassword")}</a></span>
                                        </#if>
                                    </div>
                                </div>

                                <div id="kc-form-buttons" class="${properties.kcFormButtonsClass!}">
                                    <div class="${properties.kcFormButtonsWrapperClass!}">
                                        <input tabindex="4" class="button" name="login" id="kc-login" type="submit" value="${msg("doLogIn")}"/>
                                    </div>
                                </div>
                            </div>
                        </form>
                    </#if>
                <#elseif section = "info" >
                    <#if realm.password && realm.registrationAllowed && !usernameEditDisabled??>
                        <div id="kc-registration">
                            <span>${msg("noAccount")} <a tabindex="6" href="${url.registrationUrl}" class="link">${msg("doRegister")}</a></span>
                        </div>
                    </#if>

                    <#if realm.password && social.providers??>
                    <div id="kc-social-providers">
                    <h3><a href="#social-provider-selector" data-revealer>${msg("providerLogin")}</a></h3>
                    <div id="social-provider-selector">
                    <label for="social-provider-filter">${msg("providerLoginLabel")}</label>
                    <input id="social-provider-filter" name="social-provider-filter" type="text" />
                    <ul data-filtered-by="social-provider-filter">
                        <#list social.providers as p>
                            <li title="${p.displayName}"><a href="${p.loginUrl}" id="zocial-${p.alias}" class="next-button ${p.providerId}"> <span class="text">${p.displayName}</span></a></li>
                        </#list>
                    </ul>
                    </div>
                    </div>
                    </#if>
                </#if>


            </@layout.registrationLayout>
            </div>
        </div>
        <div class="grid5">
            <div class="img"></div>
        </div>
    </div>
</div>




            
       