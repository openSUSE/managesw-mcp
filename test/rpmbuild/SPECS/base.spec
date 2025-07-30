Name:           base
Version:        1.0
Release:        1
Summary:        A base package
License:        MIT
Prefix:         /usr

%description
This is a base package.

%install
mkdir -p %{buildroot}/%{_prefix}/bin
cat << EOF > %{buildroot}/%{_prefix}/bin/base_command
#!/bin/sh
echo "Hello from base"
EOF
chmod +x %{buildroot}/%{_prefix}/bin/base_command

%files
%{_prefix}/bin/base_command